import type { ReactNode } from 'react';

interface MarkdownMessageProps {
  content: string;
}

function renderInline(text: string): ReactNode[] {
  const nodes: ReactNode[] = [];
  const pattern = /(\[([^\]]+)\]\((https?:\/\/[^\s)]+)\))|(`([^`]+)`)/g;
  let lastIndex = 0;
  let match: RegExpExecArray | null;

  while ((match = pattern.exec(text)) !== null) {
    if (match.index > lastIndex) {
      nodes.push(text.slice(lastIndex, match.index));
    }

    if (match[2] && match[3]) {
      nodes.push(
        <a
          key={`link-${match.index}`}
          href={match[3]}
          target="_blank"
          rel="noreferrer"
          className="text-primary-300 underline decoration-primary-300/40 underline-offset-2 hover:text-primary-200"
        >
          {match[2]}
        </a>
      );
    } else if (match[5]) {
      nodes.push(
        <code key={`code-${match.index}`} className="rounded bg-dark-950 px-1.5 py-0.5 font-mono text-[0.9em] text-primary-200">
          {match[5]}
        </code>
      );
    }

    lastIndex = pattern.lastIndex;
  }

  if (lastIndex < text.length) {
    nodes.push(text.slice(lastIndex));
  }

  return nodes;
}

export default function MarkdownMessage({ content }: MarkdownMessageProps) {
  const lines = content.split('\n');
  const blocks: ReactNode[] = [];
  let index = 0;

  while (index < lines.length) {
    const line = lines[index];

    if (line.trim().startsWith('```')) {
      const language = line.trim().slice(3).trim();
      const codeLines: string[] = [];
      index += 1;
      while (index < lines.length && !lines[index].trim().startsWith('```')) {
        codeLines.push(lines[index]);
        index += 1;
      }
      blocks.push(
        <pre key={`code-${index}`} className="my-3 overflow-x-auto rounded-2xl border border-dark-700 bg-dark-950 p-4 text-sm text-dark-100">
          {language ? <div className="mb-2 text-xs uppercase tracking-[0.18em] text-dark-500">{language}</div> : null}
          <code>{codeLines.join('\n')}</code>
        </pre>
      );
      index += 1;
      continue;
    }

    if (line.startsWith('### ')) {
      blocks.push(<h3 key={index} className="mt-4 text-base font-semibold text-white">{renderInline(line.slice(4))}</h3>);
    } else if (line.startsWith('## ')) {
      blocks.push(<h2 key={index} className="mt-4 text-lg font-semibold text-white">{renderInline(line.slice(3))}</h2>);
    } else if (line.startsWith('# ')) {
      blocks.push(<h1 key={index} className="mt-4 text-xl font-semibold text-white">{renderInline(line.slice(2))}</h1>);
    } else if (/^[-*]\s+/.test(line)) {
      const items: string[] = [];
      while (index < lines.length && /^[-*]\s+/.test(lines[index])) {
        items.push(lines[index].replace(/^[-*]\s+/, ''));
        index += 1;
      }
      blocks.push(
        <ul key={`ul-${index}`} className="my-3 list-disc space-y-1 pl-5 text-sm leading-6">
          {items.map((item, itemIndex) => <li key={itemIndex}>{renderInline(item)}</li>)}
        </ul>
      );
      continue;
    } else if (/^\d+\.\s+/.test(line)) {
      const items: string[] = [];
      while (index < lines.length && /^\d+\.\s+/.test(lines[index])) {
        items.push(lines[index].replace(/^\d+\.\s+/, ''));
        index += 1;
      }
      blocks.push(
        <ol key={`ol-${index}`} className="my-3 list-decimal space-y-1 pl-5 text-sm leading-6">
          {items.map((item, itemIndex) => <li key={itemIndex}>{renderInline(item)}</li>)}
        </ol>
      );
      continue;
    } else if (line.startsWith('> ')) {
      blocks.push(<blockquote key={index} className="my-3 border-l-2 border-primary-400/60 pl-3 text-sm italic text-dark-300">{renderInline(line.slice(2))}</blockquote>);
    } else if (line.trim() === '') {
      blocks.push(<div key={index} className="h-2" />);
    } else {
      blocks.push(<p key={index} className="text-sm leading-6">{renderInline(line)}</p>);
    }

    index += 1;
  }

  return <div className="space-y-1 break-words">{blocks}</div>;
}
