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

vi.mock('react-dnd-html5-backend', () => ({
  HTML5Backend: {},
}));

const createJSONResponse = (data) =>
  Promise.resolve({
    json: () => Promise.resolve(data),
  });

describe('App', () => {
  beforeEach(() => {
    dndState.players.clear();
    dndState.dropTargets.clear();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('renders the app header', async () => {
    global.fetch = vi.fn(() =>
      createJSONResponse([])
    );

    render(<App />);

    expect(screen.getByRole('heading', { name: 'Lineup Lab' })).toBeInTheDocument();
    await waitFor(() => expect(global.fetch).toHaveBeenCalledTimes(1));
  });

  it('loads teams on mount', async () => {
    global.fetch = vi.fn(() =>
      createJSONResponse([
        { name: 'Yankees', year: 2024 },
        { name: 'Mets', year: 2023 },
      ])
    );

    render(<App />);

    expect(await screen.findByRole('option', { name: 'Yankees (2024)' })).toBeInTheDocument();
    expect(await screen.findByRole('option', { name: 'Mets (2023)' })).toBeInTheDocument();
    expect(global.fetch).toHaveBeenCalledWith('/api/teams');
  });

  it('loads roster data after selecting a team', async () => {
    const user = userEvent.setup();
    global.fetch = vi
      .fn()
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
        2,
        '/api/batting?team=Yankees&year=2024'
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

    const [url, options] = global.fetch.mock.calls[2];
    expect(url).toBe('/api/simulate');
    expect(options.method).toBe('POST');
    expect(options.headers['Content-Type']).toBe('application/json');
    expect(JSON.parse(options.body)).toEqual(batters);
    expect(await screen.findByText(/Average Score:/)).toHaveTextContent('4.50');
    expect(screen.getByText(/Average Hits:/)).toHaveTextContent('9.25');
  });
});
