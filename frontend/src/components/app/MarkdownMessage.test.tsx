import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import MarkdownMessage from './MarkdownMessage';

describe('MarkdownMessage', () => {
  it('renders headings, bullets, numbered items, links, inline code, and fenced code', () => {
    render(
      <MarkdownMessage
        content={`## Plan
- First step
1. Numbered step
Use \`credits\` safely.
[Open docs](https://example.com)
\`\`\`ts
const ok = true;
\`\`\``}
      />
    );

    expect(screen.getByRole('heading', { name: 'Plan' })).toBeInTheDocument();
    expect(screen.getByText('First step')).toBeInTheDocument();
    expect(screen.getByText('Numbered step')).toBeInTheDocument();
    expect(screen.getByText('credits')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Open docs' })).toHaveAttribute('href', 'https://example.com');
    expect(screen.getByText('const ok = true;')).toBeInTheDocument();
  });

  it('does not render raw html as html', () => {
    render(<MarkdownMessage content={'<img src=x onerror=alert(1)> plain text'} />);

    expect(screen.getByText('<img src=x onerror=alert(1)> plain text')).toBeInTheDocument();
    expect(document.querySelector('img')).not.toBeInTheDocument();
  });
});
