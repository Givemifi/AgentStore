// Attachment processing utilities for the chat composer.
// Images are returned as base64 data URLs (sent to the LLM vision API).
// Documents (PDF / Word / TXT) are parsed to plain text in the browser
// and prepended to the outgoing message. PDF/Word parsers are loaded
// lazily via dynamic import so they don't bloat the initial bundle.

export type AttachmentKind = 'image' | 'document';

export interface PreparedAttachment {
  id: string;
  kind: AttachmentKind;
  name: string;
  /** For images: base64 data URL. For documents: undefined. */
  dataUrl?: string;
  /** For documents: extracted plain text. For images: undefined. */
  text?: string;
  /** Original byte size. */
  size: number;
}

export const MAX_IMAGE_BYTES = 5 * 1024 * 1024; // 5MB
export const MAX_DOC_BYTES = 2 * 1024 * 1024; // 2MB
export const MAX_ATTACHMENTS = 4;

const IMAGE_TYPES = ['image/jpeg', 'image/png', 'image/webp', 'image/gif'];

export function isImageFile(file: File): boolean {
  return IMAGE_TYPES.includes(file.type) || file.type.startsWith('image/');
}

function readAsDataURL(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result));
    reader.onerror = () => reject(new Error('读取文件失败'));
    reader.readAsDataURL(file);
  });
}

function readAsText(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result));
    reader.onerror = () => reject(new Error('读取文件失败'));
    reader.readAsText(file);
  });
}

async function extractPdfText(file: File): Promise<string> {
  const pdfjs = await import('pdfjs-dist');
  // Use a worker bundled by Vite via the ?url import.
  const workerUrl = (await import('pdfjs-dist/build/pdf.worker.min.mjs?url')).default;
  pdfjs.GlobalWorkerOptions.workerSrc = workerUrl;

  const buffer = await file.arrayBuffer();
  const doc = await pdfjs.getDocument({ data: buffer }).promise;
  const parts: string[] = [];
  const maxPages = Math.min(doc.numPages, 30); // cap for safety
  for (let i = 1; i <= maxPages; i += 1) {
    const page = await doc.getPage(i);
    const content = await page.getTextContent();
    const pageText = content.items
      .map((item) => ('str' in item ? item.str : ''))
      .join(' ');
    parts.push(pageText);
  }
  return parts.join('\n\n');
}

async function extractDocxText(file: File): Promise<string> {
  const mammoth = await import('mammoth');
  const buffer = await file.arrayBuffer();
  const result = await mammoth.extractRawText({ arrayBuffer: buffer });
  return result.value;
}

/**
 * Process a single picked File into a PreparedAttachment.
 * Throws an Error with a user-friendly Chinese message on failure.
 */
export async function prepareAttachment(file: File): Promise<PreparedAttachment> {
  const id = `${file.name}-${file.size}-${Date.now()}-${Math.random().toString(36).slice(2)}`;

  if (isImageFile(file)) {
    if (file.size > MAX_IMAGE_BYTES) {
      throw new Error(`图片「${file.name}」超过 5MB 上限`);
    }
    const dataUrl = await readAsDataURL(file);
    return { id, kind: 'image', name: file.name, dataUrl, size: file.size };
  }

  // Documents
  if (file.size > MAX_DOC_BYTES) {
    throw new Error(`文档「${file.name}」超过 2MB 上限`);
  }

  const lower = file.name.toLowerCase();
  let text = '';
  if (file.type === 'application/pdf' || lower.endsWith('.pdf')) {
    text = await extractPdfText(file);
  } else if (
    file.type === 'application/vnd.openxmlformats-officedocument.wordprocessingml.document' ||
    lower.endsWith('.docx')
  ) {
    text = await extractDocxText(file);
  } else if (file.type.startsWith('text/') || lower.endsWith('.txt') || lower.endsWith('.md')) {
    text = await readAsText(file);
  } else {
    throw new Error(`暂不支持的文件类型：${file.name}`);
  }

  text = text.trim();
  if (!text) {
    throw new Error(`无法从「${file.name}」中提取文字`);
  }

  return { id, kind: 'document', name: file.name, text, size: file.size };
}

/**
 * Build the final message text by prepending extracted document text,
 * and collect image data URLs for the attachments field.
 */
export function composeMessagePayload(
  draft: string,
  attachments: PreparedAttachment[],
): { message: string; images: string[] } {
  const docs = attachments.filter((a) => a.kind === 'document' && a.text);
  const images = attachments
    .filter((a) => a.kind === 'image' && a.dataUrl)
    .map((a) => a.dataUrl as string);

  let message = draft.trim();
  if (docs.length > 0) {
    const docBlocks = docs
      .map((d) => `【文档：${d.name}】\n${d.text}`)
      .join('\n\n');
    message = message ? `${docBlocks}\n\n---\n\n${message}` : docBlocks;
  }

  return { message, images };
}
