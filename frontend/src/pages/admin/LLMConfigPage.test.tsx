import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import axios from 'axios';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import LLMConfigPage from './LLMConfigPage';

const apiMocks = vi.hoisted(() => ({
  getLLMConfig: vi.fn(),
  updateLLMConfig: vi.fn(),
}));

const toastMocks = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
}));

vi.mock('../../api/client', () => ({
  adminApi: {
    getLLMConfig: apiMocks.getLLMConfig,
    updateLLMConfig: apiMocks.updateLLMConfig,
  },
}));

vi.mock('sonner', () => ({
  toast: toastMocks,
}));

function renderLLMConfigPage() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <LLMConfigPage />
    </QueryClientProvider>
  );
}

describe('LLMConfigPage', () => {
  beforeEach(() => {
    apiMocks.getLLMConfig.mockReset();
    apiMocks.updateLLMConfig.mockReset();
    toastMocks.success.mockReset();
    toastMocks.error.mockReset();

    apiMocks.getLLMConfig.mockResolvedValue({
      id: 'cfg-1',
      apiKey: '********',
      baseURL: 'https://api.example.com/v1',
      model: 'model-a',
      isActive: true,
    });
    apiMocks.updateLLMConfig.mockResolvedValue({});
  });

  it('shows an access denied state when the LLM config API returns 403', async () => {
    apiMocks.getLLMConfig.mockRejectedValue(
      new axios.AxiosError('Request failed', undefined, undefined, undefined, {
        status: 403,
        statusText: 'Forbidden',
        headers: {},
        config: { headers: {} as never },
        data: { error: 'Forbidden' },
      } as never)
    );

    renderLLMConfigPage();

    expect(await screen.findByText('You do not have permission to manage LLM configuration.')).toBeInTheDocument();
    expect(screen.queryByLabelText('API Key')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('Base URL')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('Model')).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /save configuration/i })).not.toBeInTheDocument();
    expect(apiMocks.updateLLMConfig).not.toHaveBeenCalled();
  });

  it('does not save blank required values', async () => {
    const user = userEvent.setup();
    renderLLMConfigPage();

    const apiKey = await screen.findByLabelText('API Key');
    await user.clear(apiKey);
    await user.clear(screen.getByLabelText('Base URL'));
    await user.clear(screen.getByLabelText('Model'));
    await user.click(screen.getByRole('button', { name: /save configuration/i }));

    expect(await screen.findByText('API Key is required.')).toBeInTheDocument();
    expect(screen.getByText('Base URL is required.')).toBeInTheDocument();
    expect(screen.getByText('Model is required.')).toBeInTheDocument();
    expect(apiMocks.updateLLMConfig).not.toHaveBeenCalled();
  });

  it('saves non-empty configuration values', async () => {
    const user = userEvent.setup();
    renderLLMConfigPage();

    const apiKey = await screen.findByLabelText('API Key');
    await user.clear(apiKey);
    await user.type(apiKey, 'sk-live');
    await user.clear(screen.getByLabelText('Base URL'));
    await user.type(screen.getByLabelText('Base URL'), 'https://llm.example.com/v1');
    await user.clear(screen.getByLabelText('Model'));
    await user.type(screen.getByLabelText('Model'), 'claude-compatible');
    await user.click(screen.getByRole('button', { name: /save configuration/i }));

    await waitFor(() => {
      expect(apiMocks.updateLLMConfig).toHaveBeenCalledWith({
        apiKey: 'sk-live',
        baseURL: 'https://llm.example.com/v1',
        model: 'claude-compatible',
        isActive: true,
      });
    });
  });
});
