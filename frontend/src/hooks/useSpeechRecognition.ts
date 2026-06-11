// useSpeechRecognition — thin wrapper around the browser-native
// Web Speech API for push-to-talk voice input. Frontend-only, zero cost.
// Returns interim + final transcripts. Unsupported browsers report
// `supported: false` so the UI can hide the mic button.

import { useCallback, useEffect, useRef, useState } from 'react';

// Minimal typings for the Web Speech API (not in lib.dom for all targets).
interface SpeechRecognitionResult {
  readonly isFinal: boolean;
  readonly length: number;
  item(index: number): { transcript: string };
  [index: number]: { transcript: string };
}
interface SpeechRecognitionEventLike {
  readonly resultIndex: number;
  readonly results: {
    readonly length: number;
    item(index: number): SpeechRecognitionResult;
    [index: number]: SpeechRecognitionResult;
  };
}
interface SpeechRecognitionLike {
  lang: string;
  continuous: boolean;
  interimResults: boolean;
  start(): void;
  stop(): void;
  abort(): void;
  onresult: ((event: SpeechRecognitionEventLike) => void) | null;
  onerror: ((event: { error: string }) => void) | null;
  onend: (() => void) | null;
}
type SpeechRecognitionCtor = new () => SpeechRecognitionLike;

function getRecognitionCtor(): SpeechRecognitionCtor | null {
  if (typeof window === 'undefined') return null;
  const w = window as unknown as {
    SpeechRecognition?: SpeechRecognitionCtor;
    webkitSpeechRecognition?: SpeechRecognitionCtor;
  };
  return w.SpeechRecognition ?? w.webkitSpeechRecognition ?? null;
}

export interface UseSpeechRecognitionResult {
  supported: boolean;
  listening: boolean;
  interim: string;
  error: string | null;
  start: () => void;
  stop: () => void;
}

export function useSpeechRecognition(
  onFinalTranscript: (text: string) => void,
  lang?: string,
): UseSpeechRecognitionResult {
  // The constructor is stable for the lifetime of the page; capture it once
  // in state so we never read a ref during render.
  const [ctor] = useState<SpeechRecognitionCtor | null>(() => getRecognitionCtor());
  const recognitionRef = useRef<SpeechRecognitionLike | null>(null);
  const [listening, setListening] = useState(false);
  const [interim, setInterim] = useState('');
  const [error, setError] = useState<string | null>(null);
  // Keep latest callback without re-creating the recognition instance.
  const onFinalRef = useRef(onFinalTranscript);
  useEffect(() => {
    onFinalRef.current = onFinalTranscript;
  }, [onFinalTranscript]);

  const supported = ctor !== null;

  const stop = useCallback(() => {
    const rec = recognitionRef.current;
    if (rec) {
      try {
        rec.stop();
      } catch {
        // ignore
      }
    }
    setListening(false);
  }, []);

  const start = useCallback(() => {
    if (!ctor) return;

    // Abort any prior instance.
    if (recognitionRef.current) {
      try {
        recognitionRef.current.abort();
      } catch {
        // ignore
      }
    }

    const rec = new ctor();
    rec.lang = lang ?? (typeof navigator !== 'undefined' ? navigator.language : 'zh-CN');
    rec.continuous = false;
    rec.interimResults = true;

    rec.onresult = (event) => {
      let finalText = '';
      let interimText = '';
      for (let i = event.resultIndex; i < event.results.length; i += 1) {
        const result = event.results[i];
        const transcript = result[0]?.transcript ?? '';
        if (result.isFinal) {
          finalText += transcript;
        } else {
          interimText += transcript;
        }
      }
      if (interimText) setInterim(interimText);
      if (finalText) {
        onFinalRef.current(finalText);
        setInterim('');
      }
    };

    rec.onerror = (event) => {
      // "no-speech" / "aborted" are expected when the user releases quickly.
      if (event.error !== 'no-speech' && event.error !== 'aborted') {
        setError(speechErrorMessage(event.error));
      }
      setListening(false);
    };

    rec.onend = () => {
      setListening(false);
      setInterim('');
    };

    recognitionRef.current = rec;
    setError(null);
    setInterim('');
    try {
      rec.start();
      setListening(true);
    } catch {
      setListening(false);
    }
  }, [ctor, lang]);

  useEffect(() => {
    return () => {
      if (recognitionRef.current) {
        try {
          recognitionRef.current.abort();
        } catch {
          // ignore
        }
      }
    };
  }, []);

  return { supported, listening, interim, error, start, stop };
}

function speechErrorMessage(error: string): string {
  switch (error) {
    case 'not-allowed':
    case 'service-not-allowed':
      return '麦克风权限被拒绝，请在浏览器设置中允许';
    case 'audio-capture':
      return '未检测到麦克风设备';
    case 'network':
      return '语音识别网络错误，请重试';
    default:
      return '语音识别出错，请重试';
  }
}
