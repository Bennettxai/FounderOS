import { describe, expect, it } from 'vitest';
import { parseChatSSEStream } from './chatSSEParser';

function streamFromText(text: string): ReadableStreamDefaultReader<Uint8Array> {
  return new ReadableStream<Uint8Array>({
    start(controller) {
      controller.enqueue(new TextEncoder().encode(text));
      controller.close();
    },
  }).getReader();
}

async function collectEvents(text: string) {
  const events = [];
  for await (const event of parseChatSSEStream(streamFromText(text))) {
    events.push(event);
  }
  return events;
}

describe('parseChatSSEStream', () => {
  it('parses backend tool call and tool result event payloads', async () => {
    const events = await collectEvents(
      [
        'event: tool_call',
        'data: {"type":"tool_call","data":{"tool_name":"create_task","tool_call_id":"call_1","params":"{\\"title\\":\\"Ship\\"}","status":"running"}}',
        '',
        'event: tool_result',
        'data: {"type":"tool_result","data":{"tool_call_id":"call_1","status":"completed","result":"created"}}',
        '',
      ].join('\n'),
    );

    expect(events).toEqual([
      {
        type: 'tool_call',
        toolName: 'create_task',
        toolCallId: 'call_1',
        params: { title: 'Ship' },
        status: 'calling',
      },
      {
        type: 'tool_result',
        toolName: '',
        toolCallId: 'call_1',
        content: 'created',
        status: 'success',
      },
    ]);
  });

  it('parses legacy chunk and end events used by chat tests', async () => {
    const events = await collectEvents(
      [
        'data: {"type":"chunk","content":"Hello "}',
        '',
        'data: {"type":"chunk","content":"there!"}',
        '',
        'data: {"type":"end"}',
        '',
      ].join('\n'),
    );

    expect(events).toEqual([
      { type: 'token', content: 'Hello ' },
      { type: 'token', content: 'there!' },
      { type: 'done' },
    ]);
  });

  it('parses artifact data emitted directly in the event payload', async () => {
    const events = await collectEvents(
      [
        'event: artifact_complete',
        'data: {"type":"artifact_complete","title":"Plan","artifactType":"document","content":"# Plan"}',
        '',
      ].join('\n'),
    );

    expect(events).toEqual([
      {
        type: 'artifact_complete',
        title: 'Plan',
        artifactType: 'document',
        content: '# Plan',
      },
    ]);
  });
});
