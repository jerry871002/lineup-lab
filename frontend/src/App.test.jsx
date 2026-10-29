import React from 'react';
import { act, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { vi } from 'vitest';
import App from './App.jsx';

const dndState = vi.hoisted(() => ({
  players: new Map(),
  dropTargets: new Map(),
}));

vi.mock('react-dnd', () => ({
  DndProvider: ({ children }) => children,
  useDrag: ({ item }) => {
    dndState.players.set(item.name, item);
    return [{ isDragging: false }, () => {}];
  },
  useDrop: ({ drop }) => [{ isOver: false }, node => {
    if (node) dndState.dropTargets.set(node, drop);
  }],
}));

const panelState = vi.hoisted(() => ({ saved: null }));
vi.mock('./SavedLineups.jsx', () => ({
  default: ({ onLoad, onSimulate, onSessionExpired }) => panelState.saved ? <div>
    <button onClick={() => onLoad(panelState.saved)}>Load test snapshot</button>
    <button onClick={() => onSimulate(panelState.saved, new AbortController().signal).catch(() => {})}>Record test snapshot</button>
    <button onClick={onSessionExpired}>Expire test session</button>
  </div> : null,
}));
vi.mock('./PersonalHistory.jsx', () => ({ default: ({ refreshToken }) => <span data-testid="history-refresh">{refreshToken}</span> }));

vi.mock('react-dnd-html5-backend', () => ({
  HTML5Backend: {},
}));

const createJSONResponse = (data) =>
  Promise.resolve({
    ok: true,
    status: 200,
    json: () => Promise.resolve(data),
  });

const createErrorResponse = (status, data) =>
  Promise.resolve({
    ok: false,
    status,
    json: () => Promise.resolve(data),
  });

const currentUser = { id: 1, username: 'testuser', email: 'test@example.com' };

const savedFixture = () => ({
  id: 42, name: 'Stored order', team: 'Yankees', year: 2024, revision: 1,
  batters: Array.from({ length: 9 }, (_, index) => ({ id: index + 1, name: `Saved Player ${index + 1}`, at_bat: 100, hit: 30, double: 5, triple: 1, home_run: 4, ball_on_base: 10, hit_by_pitch: 2 })),
});

describe('App', () => {
  beforeEach(() => {
    panelState.saved = null;
    dndState.players.clear();
    dndState.dropTargets.clear();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    document.cookie = 'lineup_lab_csrf=; Max-Age=0; path=/';
  });

  it('renders the app header', async () => {
    global.fetch = vi.fn(() =>
      createErrorResponse(401, { detail: 'authentication required' })
    );

    render(<App />);

    expect(screen.getByRole('heading', { name: 'Lineup Lab' })).toBeInTheDocument();
    await waitFor(() => expect(global.fetch).toHaveBeenCalledTimes(1));
  });

  it('shows the login form when there is no active session', async () => {
    global.fetch = vi.fn(() =>
      createErrorResponse(401, { detail: 'authentication required' })
    );

    render(<App />);

    expect(await screen.findByRole('tab', { name: 'Log in' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Log in' })).toBeInTheDocument();
    expect(screen.queryByText('Roster')).not.toBeInTheDocument();
  });

  it('logs in and loads protected workspace data', async () => {
    const user = userEvent.setup();
    global.fetch = vi
      .fn()
      .mockImplementationOnce(() =>
        createErrorResponse(401, { detail: 'authentication required' })
      )
      .mockImplementationOnce(() =>
        createJSONResponse(currentUser)
      )
      .mockImplementationOnce(() =>
        createJSONResponse([
          { name: 'Yankees', year: 2024 },
        ])
      );

    render(<App />);

    await user.type(await screen.findByLabelText('Username or email'), 'testuser');
    await user.type(screen.getByLabelText('Password'), 'correct horse battery');
    await user.click(screen.getByRole('button', { name: 'Log in' }));

    await waitFor(() => {
      expect(global.fetch).toHaveBeenNthCalledWith(
        2,
        '/api/auth/login',
        expect.objectContaining({
          method: 'POST',
          credentials: 'same-origin',
        })
      );
    });
    expect(await screen.findByText(/Signed in as/)).toBeInTheDocument();
    expect(await screen.findByRole('option', { name: 'Yankees (2024)' })).toBeInTheDocument();
  });

  it('shows readable validation errors from the auth API', async () => {
    const user = userEvent.setup();
    global.fetch = vi
      .fn()
      .mockImplementationOnce(() =>
        createErrorResponse(401, { detail: 'authentication required' })
      )
      .mockImplementationOnce(() =>
        createErrorResponse(422, {
          detail: [
            { msg: 'Password must be at least 15 characters long' },
            { msg: 'Username may only contain letters, numbers, and underscores' },
          ],
        })
      );

    render(<App />);

    await user.type(await screen.findByLabelText('Username or email'), 'bad username');
    await user.type(screen.getByLabelText('Password'), 'short');
    await user.click(screen.getByRole('button', { name: 'Log in' }));

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Password must be at least 15 characters long Username may only contain letters, numbers, and underscores'
    );
  });

  it('prevents duplicate auth submissions while a request is in flight', async () => {
    const user = userEvent.setup();
    let resolveLogin;
    const loginPromise = new Promise(resolve => {
      resolveLogin = resolve;
    });
    global.fetch = vi
      .fn()
      .mockImplementationOnce(() =>
        createErrorResponse(401, { detail: 'authentication required' })
      )
      .mockImplementationOnce(() =>
        loginPromise
      )
      .mockImplementationOnce(() =>
        createJSONResponse([])
      );

    render(<App />);

    await user.type(await screen.findByLabelText('Username or email'), 'testuser');
    await user.type(screen.getByLabelText('Password'), 'correct horse battery');
    const submitButton = screen.getByRole('button', { name: 'Log in' });
    await user.click(submitButton);
    await user.click(screen.getByRole('button', { name: 'Submitting...' }));

    expect(global.fetch).toHaveBeenCalledTimes(2);
    resolveLogin(createJSONResponse(currentUser));
    expect(await screen.findByText(/Signed in as/)).toBeInTheDocument();
  });

  it('registers a new user and signs in', async () => {
    const user = userEvent.setup();
    global.fetch = vi
      .fn()
      .mockImplementationOnce(() =>
        createErrorResponse(401, { detail: 'authentication required' })
      )
      .mockImplementationOnce(() =>
        createJSONResponse(currentUser)
      )
      .mockImplementationOnce(() =>
        createJSONResponse(currentUser)
      )
      .mockImplementationOnce(() =>
        createJSONResponse([])
      );

    render(<App />);

    await user.click(await screen.findByRole('tab', { name: 'Register' }));
    await user.type(screen.getByLabelText('Username'), 'testuser');
    await user.type(screen.getByLabelText('Email'), 'test@example.com');
    await user.type(screen.getByLabelText('Password'), 'correct horse battery');
    await user.click(screen.getByRole('button', { name: 'Create account' }));

    await waitFor(() => {
      expect(global.fetch).toHaveBeenNthCalledWith(
        2,
        '/api/auth/register',
        expect.objectContaining({
          method: 'POST',
          credentials: 'same-origin',
        })
      );
      expect(global.fetch).toHaveBeenNthCalledWith(
        3,
        '/api/auth/login',
        expect.objectContaining({
          method: 'POST',
          credentials: 'same-origin',
        })
      );
    });
    expect(await screen.findByText(/Signed in as/)).toBeInTheDocument();
  });

  it('logs out with the CSRF token and hides protected content', async () => {
    const user = userEvent.setup();
    document.cookie = 'lineup_lab_csrf=csrf-token; path=/';
    global.fetch = vi
      .fn()
      .mockImplementationOnce(() =>
        createJSONResponse(currentUser)
      )
      .mockImplementationOnce(() =>
        createJSONResponse([])
      )
      .mockImplementationOnce(() =>
        createJSONResponse({ detail: 'logout succeeded' })
      );

    render(<App />);

    await user.click(await screen.findByRole('button', { name: 'Log out' }));

    await waitFor(() => {
      expect(global.fetch).toHaveBeenNthCalledWith(
        3,
        '/api/auth/logout',
        expect.objectContaining({
          method: 'POST',
          credentials: 'same-origin',
          headers: {
            'X-CSRF-Token': 'csrf-token',
          },
        })
      );
    });
    expect(await screen.findByText('You have been logged out.')).toBeInTheDocument();
    expect(screen.queryByText('Roster')).not.toBeInTheDocument();
  });

  it('unquotes the CSRF cookie emitted by the auth backend', async () => {
    const user = userEvent.setup();
    const token = '550e8400-e29b-41d4-a716-446655440000:nonce:signature=';
    document.cookie = `lineup_lab_csrf="${token}"; path=/`;
    global.fetch = vi.fn()
      .mockImplementationOnce(() => createJSONResponse(currentUser))
      .mockImplementationOnce(() => createJSONResponse([]))
      .mockImplementationOnce(() => createJSONResponse({ detail: 'logout succeeded' }));
    render(<App />);
    await user.click(await screen.findByRole('button', { name: 'Log out' }));
    expect(global.fetch.mock.calls[2][1].headers['X-CSRF-Token']).toBe(token);
    expect(await screen.findByText('You have been logged out.')).toBeInTheDocument();
  });

  it('leaves an expired session when logout returns 401', async () => {
    const user = userEvent.setup();
    global.fetch = vi.fn()
      .mockImplementationOnce(() => createJSONResponse(currentUser))
      .mockImplementationOnce(() => createJSONResponse([]))
      .mockImplementationOnce(() => createErrorResponse(401, { detail: 'authentication required' }));
    render(<App />);
    await user.click(await screen.findByRole('button', { name: 'Log out' }));
    expect(await screen.findByRole('button', { name: 'Log in' })).toBeInTheDocument();
    expect(screen.queryByText('Roster')).not.toBeInTheDocument();
  });

  it('leaves a session whose cookies expired before logout', async () => {
    const user = userEvent.setup();
    global.fetch = vi.fn()
      .mockImplementationOnce(() => createJSONResponse(currentUser))
      .mockImplementationOnce(() => createJSONResponse([]))
      .mockImplementationOnce(() => createErrorResponse(403, { detail: 'csrf token is invalid' }))
      .mockImplementationOnce(() => createErrorResponse(401, { detail: 'authentication required' }));
    render(<App />);
    await user.click(await screen.findByRole('button', { name: 'Log out' }));
    expect(await screen.findByRole('button', { name: 'Log in' })).toBeInTheDocument();
    expect(screen.queryByText('Roster')).not.toBeInTheDocument();
  });

  it('keeps an active session when logout fails CSRF validation', async () => {
    const user = userEvent.setup();
    global.fetch = vi.fn()
      .mockImplementationOnce(() => createJSONResponse(currentUser))
      .mockImplementationOnce(() => createJSONResponse([]))
      .mockImplementationOnce(() => createErrorResponse(403, { detail: 'csrf token is invalid' }))
      .mockImplementationOnce(() => createJSONResponse(currentUser));
    render(<App />);
    await user.click(await screen.findByRole('button', { name: 'Log out' }));
    expect(await screen.findByRole('alert')).toHaveTextContent('csrf token is invalid');
    expect(screen.getByText(/Signed in as/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Log out' })).toBeEnabled();
  });

  it('sends only one logout request while it is pending', async () => {
    const user = userEvent.setup();
    let completeLogout;
    global.fetch = vi.fn()
      .mockImplementationOnce(() => createJSONResponse(currentUser))
      .mockImplementationOnce(() => createJSONResponse([]))
      .mockImplementationOnce(() => new Promise(resolve => { completeLogout = resolve; }));
    render(<App />);
    await user.dblClick(await screen.findByRole('button', { name: 'Log out' }));
    expect(screen.getByRole('button', { name: 'Logging out...' })).toBeDisabled();
    expect(global.fetch.mock.calls.filter(([url]) => url === '/api/auth/logout')).toHaveLength(1);
    await act(async () => completeLogout(await createJSONResponse({ detail: 'logout succeeded' })));
    expect(await screen.findByRole('button', { name: 'Log in' })).toBeInTheDocument();
  });

  it('ignores an obsolete StrictMode session response after login', async () => {
    const user = userEvent.setup();
    let completeOldSession;
    global.fetch = vi.fn()
      .mockImplementationOnce(() => new Promise(resolve => { completeOldSession = resolve; }))
      .mockImplementationOnce(() => createErrorResponse(401, { detail: 'authentication required' }))
      .mockImplementationOnce(() => createJSONResponse(currentUser))
      .mockImplementationOnce(() => createJSONResponse([]));
    render(<React.StrictMode><App /></React.StrictMode>);
    await user.type(await screen.findByLabelText('Username or email'), 'testuser');
    await user.type(screen.getByLabelText('Password'), 'correct horse battery');
    await user.click(screen.getByRole('button', { name: 'Log in' }));
    expect(await screen.findByText(/Signed in as/)).toBeInTheDocument();
    await act(async () => completeOldSession(await createErrorResponse(401, { detail: 'authentication required' })));
    expect(screen.getByText(/Signed in as/)).toBeInTheDocument();
  });

  it.each(['teams', 'batting', 'simulate'])('ignores a pending %s response after logout and another login', async (endpoint) => {
    const user = userEvent.setup();
    let completeOldRequest;
    let firstSession = true;
    global.fetch = vi.fn(url => {
      if (url === '/api/users/me') return createJSONResponse(currentUser);
      if (url === '/api/auth/logout') {
        firstSession = false;
        return createJSONResponse({ detail: 'logout succeeded' });
      }
      if (url === '/api/auth/login') return createJSONResponse({ ...currentUser, id: 2, username: 'anotheruser' });
      if (firstSession && (url === `/api/${endpoint}` || endpoint === 'batting' && url.startsWith('/api/batting?'))) {
        return new Promise(resolve => { completeOldRequest = resolve; });
      }
      if (url === '/api/teams') return createJSONResponse([{ name: 'Yankees', year: 2024 }]);
      return createJSONResponse([]);
    });
    if (endpoint === 'simulate') panelState.saved = savedFixture();
    render(<App />);
    if (endpoint === 'batting') {
      await screen.findByRole('option', { name: 'Yankees (2024)' });
      await user.selectOptions(screen.getByRole('combobox'), 'Yankees+2024');
    }
    if (endpoint === 'simulate') {
      await user.click(await screen.findByRole('button', { name: 'Load test snapshot' }));
      await user.click(await screen.findByRole('button', { name: 'Simulate' }));
    }
    await user.click(await screen.findByRole('button', { name: 'Log out' }));
    await user.type(await screen.findByLabelText('Username or email'), 'anotheruser');
    await user.type(screen.getByLabelText('Password'), 'correct horse battery');
    await user.click(screen.getByRole('button', { name: 'Log in' }));
    await screen.findByRole('option', { name: 'Yankees (2024)' });
    const staleData = endpoint === 'teams' ? [{ name: 'Stale team', year: 2024 }]
      : endpoint === 'batting' ? [{ name: 'Stale player', at_bat: 100, hit: 30 }]
      : { average_score: 99, average_hits: 99 };
    await act(async () => completeOldRequest(await createJSONResponse(staleData)));
    expect(screen.queryByText(/Stale/)).not.toBeInTheDocument();
    expect(screen.queryByText(/Average Score:/)).not.toBeInTheDocument();
    expect(screen.getByRole('option', { name: 'Yankees (2024)' })).toBeInTheDocument();
  });

  it('falls back to the raw CSRF cookie value when decoding fails', async () => {
    const user = userEvent.setup();
    document.cookie = 'lineup_lab_csrf=raw%token; path=/';
    global.fetch = vi
      .fn()
      .mockImplementationOnce(() =>
        createJSONResponse(currentUser)
      )
      .mockImplementationOnce(() =>
        createJSONResponse([])
      )
      .mockImplementationOnce(() =>
        createJSONResponse({ detail: 'logout succeeded' })
      );

    render(<App />);

    await user.click(await screen.findByRole('button', { name: 'Log out' }));

    await waitFor(() => {
      expect(global.fetch).toHaveBeenNthCalledWith(
        3,
        '/api/auth/logout',
        expect.objectContaining({
          headers: {
            'X-CSRF-Token': 'raw%token',
          },
        })
      );
    });
  });

  it('shows a network error when logout cannot reach the auth API', async () => {
    const user = userEvent.setup();
    document.cookie = 'lineup_lab_csrf=csrf-token; path=/';
    global.fetch = vi
      .fn()
      .mockImplementationOnce(() =>
        createJSONResponse(currentUser)
      )
      .mockImplementationOnce(() =>
        createJSONResponse([])
      )
      .mockImplementationOnce(() =>
        Promise.reject(new Error('network down'))
      );

    render(<App />);

    await user.click(await screen.findByRole('button', { name: 'Log out' }));

    expect(await screen.findByRole('alert')).toHaveTextContent('Network error. Please try again.');
    expect(screen.getByText(/Signed in as/)).toBeInTheDocument();
  });

  it('loads teams on mount', async () => {
    global.fetch = vi
      .fn()
      .mockImplementationOnce(() =>
        createJSONResponse(currentUser)
      )
      .mockImplementationOnce(() =>
        createJSONResponse([
          { name: 'Yankees', year: 2024 },
          { name: 'Mets', year: 2023 },
        ])
      );

    render(<App />);

    expect(await screen.findByRole('option', { name: 'Yankees (2024)' })).toBeInTheDocument();
    expect(await screen.findByRole('option', { name: 'Mets (2023)' })).toBeInTheDocument();
    expect(global.fetch).toHaveBeenNthCalledWith(
      2,
      '/api/teams'
    );
  });

  it('keeps teams empty when the teams API returns an error object', async () => {
    global.fetch = vi
      .fn()
      .mockImplementationOnce(() =>
        createJSONResponse(currentUser)
      )
      .mockImplementationOnce(() =>
        createJSONResponse({ detail: 'unexpected response' })
      );

    render(<App />);

    expect(await screen.findByRole('combobox')).toBeInTheDocument();
    expect(screen.queryByRole('option', { name: /Yankees/ })).not.toBeInTheDocument();
  });

  it('loads roster data after selecting a team', async () => {
    const user = userEvent.setup();
    global.fetch = vi
      .fn()
      .mockImplementationOnce(() =>
        createJSONResponse(currentUser)
      )
      .mockImplementationOnce(() =>
        createJSONResponse([
          { name: 'Yankees', year: 2024 },
        ])
      )
      .mockImplementationOnce(() =>
        createJSONResponse([
          {
            name: 'Aaron Judge',
            at_bat: 100,
            hit: 30,
            double: 5,
            triple: 0,
            home_run: 10,
            ball_on_base: 12,
            hit_by_pitch: 3,
          },
          {
            name: 'Juan Soto',
            at_bat: 100,
            hit: 28,
            double: 4,
            triple: 1,
            home_run: 8,
            ball_on_base: 15,
            hit_by_pitch: 2,
          },
        ])
      );

    render(<App />);

    const yankeesOption = await screen.findByRole('option', { name: 'Yankees (2024)' });

    await user.selectOptions(
      await screen.findByRole('combobox'),
      yankeesOption
    );

    await waitFor(() => {
      expect(global.fetch).toHaveBeenNthCalledWith(
        3,
        '/api/batting?team=Yankees&year=2024',
        expect.objectContaining({ signal: expect.anything() })
      );
    });

    expect(await screen.findByText(/Aaron Judge/)).toBeInTheDocument();
    expect(await screen.findByText(/Juan Soto/)).toBeInTheDocument();
  });

  it('sends only the simulation batter fields after building a lineup from roster data', async () => {
    const user = userEvent.setup();
    const batters = Array.from({ length: 9 }, (_, index) => ({
      name: `Player ${index + 1}`,
      at_bat: 100,
      hit: 30 + index,
      double: 5,
      triple: 1,
      home_run: 4,
      ball_on_base: 10,
      hit_by_pitch: 2,
    }));
    const roster = batters.map((batter, index) => ({
      ...batter,
      id: index + 1,
      team: 'Yankees',
      year: 2024,
    }));
    global.fetch = vi.fn()
      .mockImplementationOnce(() => createJSONResponse(currentUser))
      .mockImplementationOnce(() => createJSONResponse([{ name: 'Yankees', year: 2024 }]))
      .mockImplementationOnce(() => createJSONResponse(roster))
      .mockImplementationOnce(() => createJSONResponse({ average_score: 4.5, average_hits: 9.25 }));

    const { container } = render(<App />);
    await screen.findByRole('option', { name: 'Yankees (2024)' });
    await user.selectOptions(screen.getByRole('combobox'), 'Yankees+2024');
    await screen.findByText(/Player 1/);

    expect(dndState.players.get('Player 1')).toEqual(expect.objectContaining({
      avg: 0.3,
      obp: 42 / 112,
      slg: 0.49,
    }));

    const slots = container.querySelectorAll('.lineup-slot');
    batters.forEach((batter, index) => {
      act(() => {
        dndState.dropTargets.get(slots[index])(dndState.players.get(batter.name));
      });
    });
    await user.click(screen.getByRole('button', { name: 'Simulate' }));

    const [url, options] = global.fetch.mock.calls[3];
    expect(url).toBe('/api/simulate');
    expect(options.method).toBe('POST');
    expect(options.headers['Content-Type']).toBe('application/json');
    expect(JSON.parse(options.body)).toEqual(batters);
    expect(await screen.findByText(/Average Score:/)).toHaveTextContent('4.50');
    expect(screen.getByText(/Average Hits:/)).toHaveTextContent('9.25');
  });
  it('restores the saved batting snapshot while excluding its IDs from the current roster', async () => {
    const user = userEvent.setup();
    const saved = savedFixture();
    panelState.saved = saved;
    const reserve = { ...saved.batters[0], id: 10, name: 'Reserve player' };
    global.fetch = vi.fn(url => {
      if (url === '/api/users/me') return createJSONResponse(currentUser);
      if (url === '/api/teams') return createJSONResponse([{ name: 'Yankees', year: 2024 }]);
      return createJSONResponse([...saved.batters.map(player => ({ ...player, hit: 99 })), reserve]);
    });
    render(<App />);
    await user.click(await screen.findByRole('button', { name: 'Load test snapshot' }));
    await screen.findByText(/Reserve player/);
    expect(dndState.players.get('Saved Player 1')).toEqual(expect.objectContaining({ hit: 30, avg: 0.3 }));
    expect(screen.getAllByText(/Saved Player 1/)).toHaveLength(1);
    expect(screen.getByRole('combobox')).toHaveValue('Yankees+2024');
  });

  it('disables incomplete simulation and preserves a complete lineup after a failed request', async () => {
    const user = userEvent.setup();
    panelState.saved = savedFixture();
    global.fetch = vi.fn(url => {
      if (url === '/api/users/me') return createJSONResponse(currentUser);
      if (url === '/api/teams') return createJSONResponse([{ name: 'Yankees', year: 2024 }]);
      if (url === '/api/simulate') return createErrorResponse(400, { detail: 'invalid lineup' });
      return createJSONResponse([]);
    });
    render(<App />);
    expect(await screen.findByRole('button', { name: 'Simulate' })).toBeDisabled();
    await user.click(screen.getByRole('button', { name: 'Load test snapshot' }));
    await user.click(screen.getByRole('button', { name: 'Simulate' }));
    expect(await screen.findByRole('alert')).toHaveTextContent('Simulation failed');
    expect(screen.getByText(/Saved Player 1/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Simulate' })).toBeEnabled();
  });

  it('records a saved lineup with server-side scores and refreshes personal history', async () => {
    const user = userEvent.setup();
    const saved = savedFixture();
    panelState.saved = saved;
    document.cookie = 'lineup_lab_csrf="signed:csrf:token="; path=/';
    const run = { ...saved, lineup_name: saved.name, lineup_revision: 1, average_score: 4.5, average_hits: 9.25, sample_count: 10, simulator_version: 'lineup-engine-v1' };
    global.fetch = vi.fn(url => {
      if (url === '/api/users/me') return createJSONResponse(currentUser);
      if (url === '/api/teams') return createJSONResponse([{ name: 'Yankees', year: 2024 }]);
      if (url === '/api/lineups/42/simulations') return createJSONResponse(run);
      return createJSONResponse([]);
    });
    render(<App />);
    await user.click(await screen.findByRole('button', { name: 'Record test snapshot' }));
    expect(await screen.findByText(/Average Score:/)).toHaveTextContent('4.50');
    expect(screen.getByTestId('history-refresh')).toHaveTextContent('1');
    const [, options] = global.fetch.mock.calls.find(([url]) => url === '/api/lineups/42/simulations');
    expect(options.method).toBe('POST');
    expect(options.headers['X-CSRF-Token']).toBe('signed:csrf:token=');
    expect(options.body).toBeUndefined();
  });

  it('clears the private workspace when a resource reports an expired session', async () => {
    const user = userEvent.setup();
    panelState.saved = savedFixture();
    global.fetch = vi.fn(url => url === '/api/users/me' ? createJSONResponse(currentUser) : createJSONResponse([]));
    render(<App />);
    await user.click(await screen.findByRole('button', { name: 'Load test snapshot' }));
    await user.click(screen.getByRole('button', { name: 'Expire test session' }));
    expect(await screen.findByRole('button', { name: 'Log in' })).toBeInTheDocument();
    expect(screen.queryByText(/Saved Player/)).not.toBeInTheDocument();
    expect(screen.getByText(/Your session expired/)).toBeInTheDocument();
  });

  it('blocks duplicate simulations while a request is pending', async () => {
    const user = userEvent.setup();
    panelState.saved = savedFixture();
    let finish;
    global.fetch = vi.fn(url => {
      if (url === '/api/users/me') return createJSONResponse(currentUser);
      if (url === '/api/teams') return createJSONResponse([{ name: 'Yankees', year: 2024 }]);
      if (url === '/api/simulate') return new Promise(resolve => { finish = resolve; });
      return createJSONResponse([]);
    });
    render(<App />);
    await user.click(await screen.findByRole('button', { name: 'Load test snapshot' }));
    await user.click(screen.getByRole('button', { name: 'Simulate' }));
    const busy = screen.getByRole('button', { name: 'Simulating...' });
    expect(busy).toBeDisabled();
    await user.click(busy);
    expect(global.fetch.mock.calls.filter(([url]) => url === '/api/simulate')).toHaveLength(1);
    await act(async () => finish(await createJSONResponse({ average_score: 4.5, average_hits: 9.25 })));
    expect(screen.getByRole('button', { name: 'Simulate' })).toBeEnabled();
  });

  it('clears old results and ignores the previous roster after switching team', async () => {
    const user = userEvent.setup();
    panelState.saved = savedFixture();
    let finishOldRoster;
    global.fetch = vi.fn(url => {
      if (url === '/api/users/me') return createJSONResponse(currentUser);
      if (url === '/api/teams') return createJSONResponse([{ name: 'Yankees', year: 2024 }, { name: 'New team', year: 2024 }]);
      if (url === '/api/simulate') return createJSONResponse({ average_score: 4.5, average_hits: 9.25 });
      if (url.includes('team=Yankees')) return new Promise(resolve => { finishOldRoster = resolve; });
      return createJSONResponse([{ ...savedFixture().batters[0], id: 100, name: 'New team player' }]);
    });
    render(<App />);
    await user.click(await screen.findByRole('button', { name: 'Load test snapshot' }));
    await user.click(screen.getByRole('button', { name: 'Simulate' }));
    await screen.findByText(/Average Score:/);
    await user.selectOptions(screen.getByRole('combobox'), 'New team+2024');
    await screen.findByText(/New team player/);
    await act(async () => finishOldRoster(await createJSONResponse([{ ...savedFixture().batters[0], name: 'Stale player' }])));
    expect(screen.queryByText(/Stale player/)).not.toBeInTheDocument();
    expect(screen.queryByText(/Average Score:/)).not.toBeInTheDocument();
    expect(screen.queryByText(/Saved Player/)).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Simulate' })).toBeDisabled();
  });

  it('clears the previous result when recording a saved simulation fails', async () => {
    const user = userEvent.setup();
    panelState.saved = savedFixture();
    global.fetch = vi.fn(url => {
      if (url === '/api/users/me') return createJSONResponse(currentUser);
      if (url === '/api/teams') return createJSONResponse([{ name: 'Yankees', year: 2024 }]);
      if (url === '/api/simulate') return createJSONResponse({ average_score: 4.5, average_hits: 9.25 });
      if (url === '/api/lineups/42/simulations') return createErrorResponse(503, { detail: 'Simulator unavailable' });
      return createJSONResponse([]);
    });
    render(<App />);
    await user.click(await screen.findByRole('button', { name: 'Load test snapshot' }));
    await user.click(screen.getByRole('button', { name: 'Simulate' }));
    await screen.findByText(/Average Score:/);
    await user.click(screen.getByRole('button', { name: 'Record test snapshot' }));
    await waitFor(() => expect(screen.queryByText(/Average Score:/)).not.toBeInTheDocument());
    expect(screen.getByRole('button', { name: 'Simulate' })).toBeEnabled();
  });

});
