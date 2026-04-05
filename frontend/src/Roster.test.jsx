import { fireEvent, render, screen } from '@testing-library/react';
import Roster from './Roster.jsx';

vi.mock('./Player.jsx', () => ({
  default: ({ player }) => <div>{player.name}</div>,
}));

describe('Roster', () => {
  it('renders available teams and players, and forwards selection changes', () => {
    const onTeamChange = vi.fn();

    render(
      <Roster
        players={[
          { name: 'Aaron Judge' },
          { name: 'Juan Soto' },
        ]}
        teams={[
          { name: 'Yankees', year: 2024 },
          { name: 'Mets', year: 2023 },
        ]}
        selectedTeam=""
        onTeamChange={onTeamChange}
      />
    );

    expect(screen.getByRole('option', { name: 'Yankees (2024)' })).toBeInTheDocument();
    expect(screen.getByRole('option', { name: 'Mets (2023)' })).toBeInTheDocument();
    expect(screen.getByText('Aaron Judge')).toBeInTheDocument();
    expect(screen.getByText('Juan Soto')).toBeInTheDocument();

    fireEvent.change(screen.getByRole('combobox'), {
      target: { value: 'Yankees+2024' },
    });

    expect(onTeamChange).toHaveBeenCalledTimes(1);
  });
});
