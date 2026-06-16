import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { vi } from 'vitest';
import App from './App.jsx';

vi.mock('react-dnd', () => ({
  DndProvider: ({ children }) => children,
  useDrag: () => [{ isDragging: false }, () => {}],
  useDrop: () => [{ isOver: false }, () => {}],
}));

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

describe('App', () => {
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
        '/api/batting?team=Yankees&year=2024'
      );
    });

    expect(await screen.findByText(/Aaron Judge/)).toBeInTheDocument();
    expect(await screen.findByText(/Juan Soto/)).toBeInTheDocument();
  });
});
