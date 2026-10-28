import React from 'react';
import { act, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import SavedLineups from './SavedLineups.jsx';

const batters = Array.from({ length: 9 }, (_, index) => ({
  id: index + 1,
  name: `Player ${index + 1}`,
  at_bat: 100,
  hit: 30,
  double: 5,
  triple: 1,
  home_run: 4,
  ball_on_base: 10,
  hit_by_pitch: 2,
}));
const saved = { id: 7, name: 'My order', team: 'Demo Comets', year: 2024,
  batter_ids: batters.map(player => player.id), batters, revision: 3, updated_at: '2026-10-28T20:00:00Z' };
const response = (data, status = 200) => Promise.resolve({
  ok: status >= 200 && status < 300, status, json: () => Promise.resolve(data),
});
const props = () => ({ selectedTeam: 'Demo Comets+2024', lineup: batters, onLoad: vi.fn(), onSessionExpired: vi.fn() });
const setup = async (items = []) => {
  const user = userEvent.setup();
  global.fetch = vi.fn().mockImplementationOnce(() => response(items));
  const callbacks = props();
  const view = render(<SavedLineups {...callbacks} />);
  await waitFor(() => expect(screen.queryByText('Working on saved lineups...')).not.toBeInTheDocument());
  return { user, ...callbacks, ...view };
};
const choose = async (user) => user.selectOptions(screen.getByLabelText('Saved lineup'), '7');
const name = async (user, text = 'New order') => user.type(screen.getByLabelText('Lineup name'), text);

afterEach(() => {
  vi.restoreAllMocks();
  document.cookie = 'lineup_lab_csrf=; Max-Age=0; path=/';
});

it('lists saved team-seasons and loads the latest canonical snapshot in its saved order', async () => {
  const { user, onLoad } = await setup([saved]);
  const snapshot = { ...saved, batters: [...batters].reverse(), batter_ids: [...saved.batter_ids].reverse(), revision: 4 };
  global.fetch.mockImplementationOnce(() => response(snapshot));
  await choose(user);
  await user.click(screen.getByRole('button', { name: 'Load selected lineup' }));
  expect(onLoad).toHaveBeenCalledWith(snapshot);
  expect(global.fetch).toHaveBeenLastCalledWith('/api/lineups/7', expect.objectContaining({ method: 'GET', credentials: 'same-origin' }));
  expect(screen.getByLabelText('Lineup name')).toHaveValue('My order');
  expect(screen.getByRole('button', { name: 'Update loaded lineup' })).toBeEnabled();
});

it('creates a named lineup using only ordered IDs and the unquoted signed CSRF cookie', async () => {
  const { user } = await setup();
  const token = 'session:nonce:signature=';
  document.cookie = `lineup_lab_csrf="${token}"; path=/`;
  global.fetch.mockImplementationOnce(() => response({ ...saved, name: 'New order' }, 201));
  await name(user);
  await user.click(screen.getByRole('button', { name: 'Save as new lineup' }));
  const [url, options] = global.fetch.mock.calls[1];
  expect(url).toBe('/api/lineups');
  expect(options).toEqual(expect.objectContaining({ method: 'POST', credentials: 'same-origin', headers: {
    'X-CSRF-Token': token, 'Content-Type': 'application/json',
  } }));
  expect(JSON.parse(options.body)).toEqual({ name: 'New order', team: 'Demo Comets', year: 2024, batter_ids: saved.batter_ids });
  expect(await screen.findByRole('status')).toHaveTextContent('Saved New order.');
});

it.each([
  ['empty slots', Array(9).fill(null)],
  ['too few players', batters.slice(0, 8)],
  ['duplicate IDs', [...batters.slice(0, 8), { ...batters[8], id: 1 }]],
  ['duplicate names', [...batters.slice(0, 8), { ...batters[8], name: ' player 1 ' }]],
  ['invalid statistics', [...batters.slice(0, 8), { ...batters[8], hit: 101 }]],
  ['no out probability', batters.map(player => ({ ...player, hit: 100 }))],
])('does not save a lineup with %s', async (_label, lineup) => {
  const { user, rerender, onLoad, onSessionExpired } = await setup();
  rerender(<SavedLineups {...{ selectedTeam: 'Demo Comets+2024', lineup, onLoad, onSessionExpired }} />);
  await name(user);
  expect(screen.getByRole('button', { name: 'Save as new lineup' })).toBeDisabled();
  expect(global.fetch).toHaveBeenCalledTimes(1);
});

it('requires a selected team-season and a nonblank name', async () => {
  const { user, rerender, onLoad, onSessionExpired } = await setup();
  expect(screen.getByRole('button', { name: 'Save as new lineup' })).toBeDisabled();
  await name(user);
  rerender(<SavedLineups {...{ selectedTeam: '', lineup: batters, onLoad, onSessionExpired }} />);
  expect(screen.getByRole('button', { name: 'Save as new lineup' })).toBeDisabled();
});

it('updates the last loaded lineup using its observed revision', async () => {
  const { user } = await setup([saved]);
  document.cookie = 'lineup_lab_csrf=csrf-token; path=/';
  global.fetch.mockImplementationOnce(() => response(saved)).mockImplementationOnce(() => response({ ...saved, revision: 4 }));
  await choose(user);
  await user.click(screen.getByRole('button', { name: 'Load selected lineup' }));
  await user.click(screen.getByRole('button', { name: 'Update loaded lineup' }));
  const [url, options] = global.fetch.mock.calls[2];
  expect(url).toBe('/api/lineups/7');
  expect(options.method).toBe('PUT');
  expect(options.headers['X-CSRF-Token']).toBe('csrf-token');
  expect(JSON.parse(options.body)).toEqual({ name: saved.name, team: saved.team, year: saved.year, batter_ids: saved.batter_ids, revision: 3 });
});

it('offers a fresh snapshot reload after a revision conflict without automatically overwriting edits', async () => {
  const { user, onLoad } = await setup([saved]);
  global.fetch.mockImplementationOnce(() => response(saved))
    .mockImplementationOnce(() => response({ detail: 'lineup changed; reload before saving' }, 409))
    .mockImplementationOnce(() => response({ ...saved, revision: 4, name: 'Latest order' }));
  await choose(user);
  await user.click(screen.getByRole('button', { name: 'Load selected lineup' }));
  await user.click(screen.getByRole('button', { name: 'Update loaded lineup' }));
  expect(await screen.findByRole('alert')).toHaveTextContent('lineup changed');
  expect(onLoad).toHaveBeenCalledTimes(1);
  await user.click(screen.getByRole('button', { name: 'Reload saved lineup' }));
  expect(onLoad).toHaveBeenLastCalledWith(expect.objectContaining({ revision: 4, name: 'Latest order' }));
});

it('deletes the selected saved lineup with CSRF and removes it from the list', async () => {
  const { user } = await setup([saved]);
  document.cookie = 'lineup_lab_csrf=raw%token; path=/';
  global.fetch.mockImplementationOnce(() => response(null, 204));
  await choose(user);
  await user.click(screen.getByRole('button', { name: 'Delete selected lineup' }));
  expect(global.fetch).toHaveBeenLastCalledWith('/api/lineups/7', expect.objectContaining({
    method: 'DELETE', headers: { 'X-CSRF-Token': 'raw%token' },
  }));
  expect(screen.queryByRole('option', { name: /My order/ })).not.toBeInTheDocument();
});

it('reports an expired session and clears private saved items on a write returning 401', async () => {
  const { user, onSessionExpired } = await setup([saved]);
  global.fetch.mockImplementationOnce(() => response({ detail: 'authentication required' }, 401));
  await name(user);
  await user.click(screen.getByRole('button', { name: 'Save as new lineup' }));
  expect(onSessionExpired).toHaveBeenCalledTimes(1);
  expect(await screen.findByRole('alert')).toHaveTextContent('session expired');
  expect(screen.queryByRole('option', { name: /My order/ })).not.toBeInTheDocument();
});

it('reports initial session expiration', async () => {
  const callbacks = props();
  global.fetch = vi.fn(() => response({ detail: 'authentication required' }, 401));
  render(<SavedLineups {...callbacks} />);
  expect(await screen.findByRole('alert')).toHaveTextContent('session expired');
  expect(callbacks.onSessionExpired).toHaveBeenCalledTimes(1);
});

it('keeps local input and loaded lineup on a rejected write', async () => {
  const { user, onLoad, onSessionExpired } = await setup([saved]);
  global.fetch.mockImplementationOnce(() => response({ detail: 'csrf token is invalid' }, 403));
  await name(user);
  await user.click(screen.getByRole('button', { name: 'Save as new lineup' }));
  expect(await screen.findByRole('alert')).toHaveTextContent('csrf token is invalid');
  expect(screen.getByLabelText('Lineup name')).toHaveValue('New order');
  expect(onLoad).not.toHaveBeenCalled();
  expect(onSessionExpired).not.toHaveBeenCalled();
});

it('prevents duplicate writes while saving', async () => {
  const { user } = await setup();
  let complete;
  global.fetch.mockImplementationOnce(() => new Promise(resolve => { complete = resolve; }));
  await name(user);
  await user.dblClick(screen.getByRole('button', { name: 'Save as new lineup' }));
  expect(global.fetch).toHaveBeenCalledTimes(2);
  expect(screen.getByRole('button', { name: 'Save as new lineup' })).toBeDisabled();
  await act(async () => complete(await response(saved, 201)));
});

it('aborts a pending snapshot load on unmount and ignores a late response', async () => {
  const { user, unmount, onLoad } = await setup([saved]);
  let complete;
  global.fetch.mockImplementationOnce(() => new Promise(resolve => { complete = resolve; }));
  await choose(user);
  await user.click(screen.getByRole('button', { name: 'Load selected lineup' }));
  const signal = global.fetch.mock.calls[1][1].signal;
  unmount();
  expect(signal.aborted).toBe(true);
  await act(async () => complete(await response(saved)));
  expect(onLoad).not.toHaveBeenCalled();
});

it('shows network errors and lets the user retry loading the list', async () => {
  global.fetch = vi.fn().mockRejectedValueOnce(new Error('Network unavailable')).mockImplementationOnce(() => response([saved]));
  render(<SavedLineups {...props()} />);
  expect(await screen.findByRole('alert')).toHaveTextContent('Network unavailable');
  await userEvent.click(screen.getByRole('button', { name: 'Refresh saved lineups' }));
  expect(await screen.findByRole('option', { name: /My order/ })).toBeInTheDocument();
});
