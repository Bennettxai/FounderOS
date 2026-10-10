import { browser } from "$app/environment";

type MessageEntry = { role: "user" | "assistant"; content: string };

export interface ElectronMeetingState {
  meetingSession: unknown;
  systemMediaRecorder: MediaRecorder | null;
  systemAudioChunks: Blob[];
}

export interface ElectronStateSetters {
  setIsMeetingMode: (v: boolean) => void;
  setMessages: (updater: (prev: MessageEntry[]) => MessageEntry[]) => void;
  getMeetingSession: () => unknown;
  setMeetingSession: (v: unknown) => void;
  getSystemMediaRecorder: () => MediaRecorder | null;
  setSystemMediaRecorder: (v: MediaRecorder | null) => void;
  getSystemAudioChunks: () => Blob[];
  setSystemAudioChunks: (v: Blob[]) => void;
  getUpcomingMeeting: () => { id: string; title: string; start: string } | null;
  setPendingScreenshot: (v: string | null) => void;
  setIsCapturingScreenshot: (v: boolean) => void;
  getIsCapturingScreenshot: () => boolean;
  scrollToBottom: () => void;
  hidePopup: () => void;
}

function addMessage(s: ElectronStateSetters, msg: MessageEntry) {
  s.setMessages((prev) => [...prev, msg]);
}

export async function startMeetingRecording(
  s: ElectronStateSetters,
): Promise<void> {
  s.setIsMeetingMode(true);

  if (browser && "electron" in window) {
    const electron = (
      window as unknown as { electron: Record<string, unknown> }
    ).electron;

    try {
      const meeting = electron.meeting as
        | Record<string, (args: unknown) => Promise<unknown>>
        | undefined;
      const session = await meeting?.start({
        title: s.getUpcomingMeeting()?.title || "Meeting Recording",
        calendarEventId: s.getUpcomingMeeting()?.id,
      });
      s.setMeetingSession(session);

      const constraints: MediaStreamConstraints = {
        audio: {
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          mandatory: { chromeMediaSource: "desktop" },
        } as unknown as MediaTrackConstraints,
        video: {
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          mandatory: {
            chromeMediaSource: "desktop",
            maxWidth: 1,
            maxHeight: 1,
          },
        } as unknown as MediaTrackConstraints,
      };

      const stream = await navigator.mediaDevices.getUserMedia(constraints);
      stream.getVideoTracks().forEach((track) => track.stop());

      s.setSystemAudioChunks([]);
      const mediaRecorder = new MediaRecorder(stream, {
        mimeType: "audio/webm;codecs=opus",
      });

      mediaRecorder.ondataavailable = (event) => {
        if (event.data.size > 0) {
          s.setSystemAudioChunks([...s.getSystemAudioChunks(), event.data]);
        }
      };

      mediaRecorder.onstop = async () => {
        const audioBlob = new Blob(s.getSystemAudioChunks(), {
          type: "audio/webm",
        });
        const arrayBuffer = await audioBlob.arrayBuffer();
        const sessionId = (s.getMeetingSession() as { id?: string } | null)?.id;
        if (sessionId) {
          await meeting?.saveAudioChunk({
            sessionId,
            chunk: arrayBuffer,
            isLast: true,
          });
        }
        stream.getTracks().forEach((track) => track.stop());
      };

      mediaRecorder.start(10000);
      s.setSystemMediaRecorder(mediaRecorder);

      const title = s.getUpcomingMeeting()?.title;
      addMessage(s, {
        role: "assistant",
        content: `Meeting recording started${title ? ` for "${title}"` : ""}. I'm capturing system audio and will transcribe when you stop. Click the mic to add voice notes, or type questions.`,
      });
    } catch {
      addMessage(s, {
        role: "assistant",
        content:
          "Could not start system audio capture. Please grant screen/audio permissions in System Preferences > Security & Privacy > Privacy > Screen Recording.",
      });
      s.setIsMeetingMode(false);
    }
  } else {
    const title = s.getUpcomingMeeting()?.title;
    addMessage(s, {
      role: "assistant",
      content: `Meeting mode started${title ? ` for "${title}"` : ""}. System audio capture requires the desktop app. Using microphone only.`,
    });
  }
}

export async function stopMeetingRecording(
  s: ElectronStateSetters,
): Promise<void> {
  s.setIsMeetingMode(false);

  const recorder = s.getSystemMediaRecorder();
  if (recorder && recorder.state !== "inactive") {
    recorder.stop();
  }

  if (browser && "electron" in window && s.getMeetingSession()) {
    const electron = (
      window as unknown as { electron: Record<string, unknown> }
    ).electron;
    const meeting = electron.meeting as
      | Record<string, () => Promise<void>>
      | undefined;
    await meeting?.stop();
  }

  addMessage(s, {
    role: "assistant",
    content:
      "Meeting recording stopped. Processing audio and generating transcription...",
  });

  s.setMeetingSession(null);
  s.setSystemMediaRecorder(null);
  s.setSystemAudioChunks([]);

  setTimeout(() => {
    addMessage(s, {
      role: "assistant",
      content:
        "Transcription and summary will be available in the full app. Open the main window to view meeting notes.",
    });
  }, 2000);
}

export async function captureScreenshot(
  s: ElectronStateSetters,
): Promise<void> {
  if (s.getIsCapturingScreenshot()) return;
  s.setIsCapturingScreenshot(true);

  try {
    if (browser && "electron" in window) {
      const electron = (
        window as unknown as { electron: Record<string, unknown> }
      ).electron;
      s.hidePopup();
      await new Promise((resolve) => setTimeout(resolve, 200));

      const screenshot = electron.screenshot as
        | Record<
            string,
            () => Promise<{
              success: boolean;
              dataUrl?: string;
              size?: { width: number; height: number };
              error?: string;
            }>
          >
        | undefined;
      const result = await screenshot?.capture();

      if (result?.success && result.dataUrl) {
        s.setPendingScreenshot(result.dataUrl);
        addMessage(s, {
          role: "user",
          content: `[Screenshot captured - ${result.size?.width}x${result.size?.height}]`,
        });
        addMessage(s, {
          role: "assistant",
          content:
            "Screenshot captured! You can describe what you want me to help with regarding this image.",
        });
      } else {
        addMessage(s, {
          role: "assistant",
          content: `Screenshot capture failed: ${result?.error || "Unknown error"}. Make sure BusinessOS has screen recording permission in System Preferences → Privacy & Security → Screen Recording.`,
        });
      }
    } else {
      try {
        const items = await navigator.clipboard.read();
        for (const item of items) {
          if (item.types.includes("image/png")) {
            const blob = await item.getType("image/png");
            const reader = new FileReader();
            reader.onload = () => {
              s.setPendingScreenshot(reader.result as string);
              addMessage(s, {
                role: "user",
                content: "[Image pasted from clipboard]",
              });
            };
            reader.readAsDataURL(blob);
            return;
          }
        }
        addMessage(s, {
          role: "assistant",
          content:
            "No image found in clipboard. Take a screenshot (Cmd+Shift+4 on Mac) and paste it here (Cmd+V).",
        });
      } catch {
        addMessage(s, {
          role: "assistant",
          content:
            "Clipboard access requires permission. Take a screenshot and paste it, or use the desktop app for direct screenshot capture.",
        });
      }
    }
  } catch {
    addMessage(s, {
      role: "assistant",
      content:
        "Screenshot capture failed. Please check screen recording permissions.",
    });
  } finally {
    s.setIsCapturingScreenshot(false);
    s.scrollToBottom();
  }
}
