import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Lineup from './Lineup.jsx';

vi.mock('./LineupSlot.jsx', () => ({
  default: ({ index }) => <div data-testid="lineup-slot">Slot {index + 1}</div>,
}));

describe('Lineup', () => {
  it('renders nine lineup slots and calls simulate handler', async () => {
    const user = userEvent.setup();
    const simulateLineup = vi.fn();

    render(
      <Lineup
        lineup={Array(9).fill(null)}
        movePlayerToSlot={vi.fn()}
        removePlayerFromSlot={vi.fn()}
        simulateLineup={simulateLineup}
        simulationResult={null}
      />
    );

    expect(screen.getAllByTestId('lineup-slot')).toHaveLength(9);

    await user.click(screen.getByRole('button', { name: 'Simulate' }));

    expect(simulateLineup).toHaveBeenCalledTimes(1);
  });

  it('shows formatted simulation results', () => {
    render(
      <Lineup
        lineup={Array(9).fill(null)}
        movePlayerToSlot={vi.fn()}
        removePlayerFromSlot={vi.fn()}
        simulateLineup={vi.fn()}
        simulationResult={{ average_score: 4.125, average_hits: 8.5 }}
      />
    );

    expect(screen.getByText(/Average Score:/)).toHaveTextContent('Average Score: 4.13');
    expect(screen.getByText(/Average Hits:/)).toHaveTextContent('Average Hits: 8.50');
  });
});
