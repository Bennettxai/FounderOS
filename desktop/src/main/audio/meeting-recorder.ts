import { desktopCapturer, ipcMain, BrowserWindow } from 'electron';
import { EventEmitter } from 'events';
import path from 'path';
import fs from 'fs';
import { app } from 'electron';

// Meeting recorder states
export type RecorderState = 'idle' | 'recording' | 'paused' | 'processing';

export interface MeetingSession {
  id: string;
  title: string;
  startTime: Date;
  endTime?: Date;
  audioPath?: string;
  transcriptionPath?: string;
  state: RecorderState;
  calendarEventId?: string;
}

export class MeetingRecorder extends EventEmitter {
  private sessions: Map<string, MeetingSession> = new Map();
  private activeSessionId: string | null = null;
  private recordingsDir: string;

  constructor() {
    super();
    this.recordingsDir = path.join(app.getPath('userData'), 'recordings');
    this.ensureRecordingsDir();
    this.setupIpcHandlers();
  }

  private ensureRecordingsDir(): void {
    if (!fs.existsSync(this.recordingsDir)) {
      fs.mkdirSync(this.recordingsDir, { recursive: true });
    }
  }

  private setupIpcHandlers(): void {
    // Get desktop capturer sources (for screen + audio capture)
    ipcMain.handle('meeting:get-sources', async () => {
      const sources = await desktopCapturer.getSources({
        types: ['window', 'screen'],
        thumbnailSize: { width: 150, height: 150 }
      });

      return sources.map(source => ({
        id: source.id,
        name: source.name,
        thumbnail: source.thumbnail.toDataURL()
      }));
    });

    // Start recording
    ipcMain.handle('meeting:start', async (_event, options: {
      title?: string;
      calendarEventId?: string;
    }) => {
      const sessionId = this.generateSessionId();
      const session: MeetingSession = {
        id: sessionId,
        title: options.title || 'Meeting Recording',
        startTime: new Date(),
        state: 'recording',
        calendarEventId: options.calendarEventId
      };

      this.sessions.set(sessionId, session);
      this.activeSessionId = sessionId;

      this.emit('recording:started', session);
      return session;
    });

    // Stop recording
    ipcMain.handle('meeting:stop', async () => {
      if (!this.activeSessionId) {
        return { error: 'No active recording' };
      }

      const session = this.sessions.get(this.activeSessionId);
      if (!session) {
        return { error: 'Session not found' };
      }

      session.endTime = new Date();
      session.state = 'processing';

      this.emit('recording:stopped', session);
      this.activeSessionId = null;

      return session;
    });

    // Pause/resume recording
    ipcMain.handle('meeting:pause', async () => {
      if (!this.activeSessionId) {
        return { error: 'No active recording' };
      }

      const session = this.sessions.get(this.activeSessionId);
      if (!session) {
        return { error: 'Session not found' };
      }

      session.state = session.state === 'paused' ? 'recording' : 'paused';
      this.emit('recording:state-change', session);
      return session;
    });

    // Get active session
    ipcMain.handle('meeting:get-active', () => {
      if (!this.activeSessionId) {
        return null;
      }
      return this.sessions.get(this.activeSessionId);
    });

    // Get all sessions
    ipcMain.handle('meeting:get-sessions', () => {
      return Array.from(this.sessions.values());
    });

    // Save audio chunk from renderer
    ipcMain.handle('meeting:save-audio-chunk', async (_event, data: {
      sessionId: string;
      chunk: ArrayBuffer;
      isLast: boolean;
    }) => {
      const session = this.sessions.get(data.sessionId);
      if (!session) {
        return { error: 'Session not found' };
      }

      const audioPath = path.join(this.recordingsDir, `${data.sessionId}.webm`);

      // Append chunk to file
      const buffer = Buffer.from(data.chunk);
      fs.appendFileSync(audioPath, buffer);

      if (data.isLast) {
        session.audioPath = audioPath;
        session.state = 'idle';
        this.emit('recording:saved', session);
      }

      return { success: true, path: audioPath };
    });

    // Get recording path
    ipcMain.handle('meeting:get-recording-path', (_, sessionId: string) => {
      const session = this.sessions.get(sessionId);
      return session?.audioPath || null;
    });
  }

  private generateSessionId(): string {
    return `meeting_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  getRecordingsDir(): string {
    return this.recordingsDir;
  }

  isRecording(): boolean {
    return this.activeSessionId !== null;
  }

  getActiveSession(): MeetingSession | null {
    if (!this.activeSessionId) return null;
    return this.sessions.get(this.activeSessionId) || null;
  }
}

// Singleton instance
let recorderInstance: MeetingRecorder | null = null;

export function initializeMeetingRecorder(): MeetingRecorder {
  if (!recorderInstance) {
    recorderInstance = new MeetingRecorder();
  }
  return recorderInstance;
}

export function getMeetingRecorder(): MeetingRecorder | null {
  return recorderInstance;
}
