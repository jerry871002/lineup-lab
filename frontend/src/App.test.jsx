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
    json: () => Promise.resolve(data),
  });

describe('App', () => {
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
    expect(global.fetch).toHaveBeenCalledWith('http://localhost:8082/teams/');
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

    await user.selectOptions(
      await screen.findByRole('combobox'),
      'Yankees+2024'
    );

    await waitFor(() => {
      expect(global.fetch).toHaveBeenNthCalledWith(
        2,
        'http://localhost:8082/batting/?team=Yankees&year=2024'
      );
    });

    expect(await screen.findByText(/Aaron Judge/)).toBeInTheDocument();
    expect(await screen.findByText(/Juan Soto/)).toBeInTheDocument();
  });
});
