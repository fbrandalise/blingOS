/**
 * Shared, ref-counted EventSource for the broker /events SSE stream.
 *
 * HTTP/1.1 browsers allow at most 6 connections per origin. The wiki
 * article view was opening 5 independent EventSources (useBrokerEvents,
 * subscribeSectionsUpdated, two subscribeEditLog, subscribePamEvents),
 * leaving only 1 slot for regular fetch requests and causing them to
 * queue as "pending" indefinitely.
 *
 * This module keeps a single underlying EventSource alive as long as
 * there is at least one subscriber, and closes it when the last
 * subscriber unsubscribes. All callers share the same TCP connection.
 */

import { sseURL } from "./client";

interface SharedSource {
  source: EventSource;
  refs: number;
}

let shared: SharedSource | null = null;

function acquire(): EventSource {
  const ES = (globalThis as { EventSource?: typeof EventSource }).EventSource;
  if (!ES) throw new Error("EventSource not available");
  if (!shared || shared.source.readyState === EventSource.CLOSED) {
    shared = { source: new ES(sseURL("/events")), refs: 0 };
  }
  shared.refs++;
  return shared.source;
}

function release(): void {
  if (!shared) return;
  shared.refs--;
  if (shared.refs <= 0) {
    shared.source.close();
    shared = null;
  }
}

/**
 * Subscribe to a named event on the shared broker /events SSE stream.
 * Returns an unsubscribe function — call it in useEffect cleanup.
 */
export function subscribeBrokerEvent(
  eventName: string,
  handler: (ev: MessageEvent) => void,
): () => void {
  let source: EventSource | null = null;
  try {
    source = acquire();
    source.addEventListener(eventName, handler as EventListener);
  } catch {
    return () => {};
  }
  const captured = source;
  return () => {
    captured.removeEventListener(eventName, handler as EventListener);
    release();
  };
}

/**
 * Subscribe to multiple named events at once on the shared stream.
 * More efficient than calling subscribeBrokerEvent repeatedly when you
 * need several event types from one component (avoids N ref increments).
 */
export function subscribeBrokerEvents(
  handlers: Record<string, (ev: MessageEvent) => void>,
): () => void {
  let source: EventSource | null = null;
  const names = Object.keys(handlers);
  if (names.length === 0) return () => {};
  try {
    source = acquire();
    for (const name of names) {
      source.addEventListener(name, handlers[name] as EventListener);
    }
  } catch {
    return () => {};
  }
  const captured = source;
  return () => {
    for (const name of names) {
      captured.removeEventListener(name, handlers[name] as EventListener);
    }
    release();
  };
}
