import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import CreditExplainer from './CreditExplainer';

describe('CreditExplainer', () => {
  it('renders balance, per-message cost, and safety copy', () => {
    render(<CreditExplainer balance={1200} perMessageCost={3} />);

    expect(screen.getByText('1,200')).toBeInTheDocument();
    expect(screen.getByText('credits available')).toBeInTheDocument();
    expect(screen.getByText('Most selected Agents cost 3 credits/message.')).toBeInTheDocument();
    expect(screen.getByText('Credits are charged only after a successful response.')).toBeInTheDocument();
  });

  it('renders singular per-message cost for one credit', () => {
    render(<CreditExplainer balance={1200} />);

    expect(screen.getByText('Most selected Agents cost 1 credit/message.')).toBeInTheDocument();
  });

  it('renders singular balance copy for one available credit', () => {
    render(<CreditExplainer balance={1} />);

    expect(screen.getByText('credit available')).toBeInTheDocument();
  });

  it('renders loading state when balance is unknown', () => {
    render(<CreditExplainer balance={null} perMessageCost={1} />);

    expect(screen.getByText('Credits will appear when usage data is available.')).toBeInTheDocument();
  });
});
