import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { vi } from 'vitest';
import PersonalHistory from './PersonalHistory.jsx';

const run = { id: 1, lineup_name: 'Opening order', team: 'Demo', year: 2024, lineup_revision: 2, average_score: 4.5, average_hits: 9.25, sample_count: 100000, simulator_version: 'lineup-engine-v1', created_at: '2026-10-28T20:00:00Z', batters: [{ name: 'Snapshot Batter' }] };
const response = data => Promise.resolve({ ok: true, status: 200, json: async () => data });

describe('PersonalHistory', () => {
  afterEach(() => vi.restoreAllMocks());

  it('shows server-recorded results, sample count, version and batting snapshot', async () => {
    global.fetch = vi.fn(() => response([run]));
    const user = userEvent.setup();
    render(<PersonalHistory />);
    expect(await screen.findByRole('heading', { name: 'Opening order' })).toBeInTheDocument();
    expect(screen.getByText('4.50')).toBeInTheDocument();
    expect(screen.getByText(/100,000 games/)).toHaveTextContent('lineup-engine-v1');
    await user.click(screen.getByText('View recorded batting order'));
    expect(screen.getByText('Snapshot Batter')).toBeVisible();
  });

  it('clears private results and leaves the workspace on an expired session', async () => {
    global.fetch = vi.fn(() => Promise.resolve({ ok: false, status: 401 }));
    const expired = vi.fn();
    render(<PersonalHistory onSessionExpired={expired} />);
    await waitFor(() => expect(expired).toHaveBeenCalledTimes(1));
    expect(screen.queryByText('Opening order')).not.toBeInTheDocument();
  });

  it('supports retry after a network failure and refresh after a new run', async () => {
    global.fetch = vi.fn().mockRejectedValueOnce(new Error('Network offline')).mockImplementationOnce(() => response([])).mockImplementationOnce(() => response([run]));
    const user = userEvent.setup();
    const { rerender } = render(<PersonalHistory refreshToken={0} />);
    expect(await screen.findByRole('alert')).toHaveTextContent('Network offline');
    await user.click(screen.getByRole('button', { name: 'Retry history' }));
    expect(await screen.findByText(/Run a simulation to start/)).toBeInTheDocument();
    rerender(<PersonalHistory refreshToken={1} />);
    expect(await screen.findByText('Opening order')).toBeInTheDocument();
  });

  it('aborts a pending read when unmounted', () => {
    global.fetch = vi.fn(() => new Promise(() => {}));
    const { unmount } = render(<PersonalHistory />);
    const signal = global.fetch.mock.calls[0][1].signal;
    unmount();
    expect(signal.aborted).toBe(true);
  });
  it('shows a current-lineup run without inventing a saved revision', async () => {
    const current = { ...run, lineup_name: 'Current lineup' };
    delete current.lineup_revision;
    global.fetch = vi.fn(() => response([current]));
    render(<PersonalHistory />);
    expect(await screen.findByRole('heading', { name: 'Current lineup' })).toBeInTheDocument();
    expect(screen.queryByText(/Saved revision/)).not.toBeInTheDocument();
    expect(screen.getByText('4.50')).toBeInTheDocument();
  });

});
