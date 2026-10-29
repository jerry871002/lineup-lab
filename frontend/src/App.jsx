import React, { useState, useEffect, useRef } from 'react';
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import Lineup from './Lineup.jsx';
import Roster from './Roster.jsx';
import SavedLineups from './SavedLineups.jsx';
import PersonalHistory from './PersonalHistory.jsx';
import './App.css';

const apiBasePath = '/api';
const csrfCookieName = 'lineup_lab_csrf';

const readCookie = (name) => {
  const prefix = `${name}=`;
  const cookie = document.cookie
    .split('; ')
    .find(value => value.startsWith(prefix));

  if (!cookie) {
    return null;
  }

  const rawValue = cookie.slice(prefix.length);
  const value = rawValue.startsWith('"') && rawValue.endsWith('"')
    ? rawValue.slice(1, -1)
    : rawValue;
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
};

const formatAPIErrorDetail = (detail) => {
  if (typeof detail === 'string') {
    return detail;
  }

  if (Array.isArray(detail)) {
    return detail
      .map(error => {
        if (typeof error === 'string') {
          return error;
        }

        if (error && typeof error === 'object' && typeof error.msg === 'string') {
          return error.msg;
        }

        return 'Invalid input';
      })
      .join(' ');
  }

  return 'Request failed';
};

const parseAPIError = async (response) => {
  try {
    const payload = await response.json();
    return formatAPIErrorDetail(payload.detail);
  } catch {
    return 'Request failed';
  }
};

const decoratePlayer = player => ({
  ...player,
  avg: player.at_bat > 0 ? player.hit / player.at_bat : 0,
  obp: player.at_bat + player.ball_on_base + player.hit_by_pitch > 0
    ? (player.hit + player.ball_on_base + player.hit_by_pitch) / (player.at_bat + player.ball_on_base + player.hit_by_pitch) : 0,
  slg: player.at_bat > 0 ? (player.hit + player.double + 2 * player.triple + 3 * player.home_run) / player.at_bat : 0,
});

const App = () => {
  const [currentUser, setCurrentUser] = useState(null);
  const [authStatus, setAuthStatus] = useState('loading');
  const [authMode, setAuthMode] = useState('login');
  const [authForm, setAuthForm] = useState({
    username: '',
    email: '',
    password: '',
  });
  const [authError, setAuthError] = useState('');
  const [authMessage, setAuthMessage] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isLoggingOut, setIsLoggingOut] = useState(false);
  const logoutPending = useRef(false);
  const workspaceVersion = useRef(0);
  const [teams, setTeams] = useState([]);
  const [teamsLoading, setTeamsLoading] = useState(false);
  const [teamsError, setTeamsError] = useState('');
  const [teamsReload, setTeamsReload] = useState(0);
  const [selectedTeam, setSelectedTeamState] = useState('');
  const selectedTeamRef = useRef('');
  const setSelectedTeam = value => { selectedTeamRef.current = value; setSelectedTeamState(value); };
  const [players, setPlayers] = useState([]);
  const [lineup, setLineupState] = useState(Array(9).fill(null));
  const lineupRef = useRef(lineup);
  const setLineup = value => { lineupRef.current = value; setLineupState(value); };
  const [rosterReload, setRosterReload] = useState(0);
  const [rosterLoading, setRosterLoading] = useState(false);
  const [rosterError, setRosterError] = useState('');
  const [isSimulating, setIsSimulating] = useState(false);
  const [simulationError, setSimulationError] = useState('');
  const [historyRefresh, setHistoryRefresh] = useState(0);
  const simulationRequest = useRef(null);
  const simulationVersion = useRef(0);
  const [simulationResult, setSimulationResult] = useState(null);

  useEffect(() => {
    let active = true;
    fetch(`${apiBasePath}/users/me`, {
      credentials: 'same-origin',
    })
      .then(async response => {
        if (response.ok) {
          const user = await response.json();
          if (active) {
            setCurrentUser(user);
            setAuthStatus('authenticated');
          }
          return;
        }

        if (active) {
          setCurrentUser(null);
          setAuthStatus('anonymous');
        }
      })
      .catch(error => {
        console.error('Error checking session:', error);
        if (active) {
          setCurrentUser(null);
          setAuthStatus('anonymous');
        }
      });
    return () => { active = false; };
  }, []);

  useEffect(() => {
    if (authStatus !== 'authenticated') {
      return;
    }

    let active = true;
    setTeamsLoading(true);
    setTeamsError('');
    fetch(`${apiBasePath}/teams`)
      .then(response => {
        if (!response.ok) {
          throw new Error('Failed to fetch teams');
        }

        return response.json();
      })
      .then(data => {
        if (!active) return;
        const available = Array.isArray(data) ? [...data] : [];
        const key = selectedTeamRef.current;
        if (key) {
          const separator = key.lastIndexOf('+');
          const name = key.slice(0, separator);
          const year = Number(key.slice(separator + 1));
          if (!available.some(team => team.name === name && team.year === year)) available.push({ name, year });
        }
        setTeams(available);
      })
      .catch(() => { if (active) setTeamsError('Could not load teams. Please try again.'); })
      .finally(() => { if (active) setTeamsLoading(false); });
    return () => { active = false; };
  }, [authStatus, teamsReload]);

  useEffect(() => {
    const controller = new AbortController();
    const version = workspaceVersion.current;
    if (authStatus === 'authenticated' && selectedTeam) {
      const separator = selectedTeam.lastIndexOf('+');
      const name = selectedTeam.slice(0, separator);
      const year = selectedTeam.slice(separator + 1);
      setRosterLoading(true);
      setRosterError('');
      fetch(`${apiBasePath}/batting?${new URLSearchParams({ team: name, year })}`, { signal: controller.signal })
        .then(async response => {
          if (!response.ok) throw new Error('Could not load the roster. Please choose the team again.');
          const data = await response.json();
          if (!Array.isArray(data)) throw new Error('The roster response could not be read.');
          if (controller.signal.aborted || version !== workspaceVersion.current) return;
          const occupied = new Set(lineupRef.current.filter(Boolean).map(player => player.id ?? player.name));
          setPlayers(data.map(decoratePlayer).filter(player => !occupied.has(player.id ?? player.name)).sort((a, b) => a.name.localeCompare(b.name)));
        })
        .catch(error => {
          if (!controller.signal.aborted && version === workspaceVersion.current) setRosterError(error.message);
        })
        .finally(() => {
          if (!controller.signal.aborted && version === workspaceVersion.current) setRosterLoading(false);
        });
    }
    return () => controller.abort();
  }, [authStatus, selectedTeam, rosterReload]);

  const cancelSimulation = () => {
    simulationVersion.current += 1;
    simulationRequest.current?.abort();
    simulationRequest.current = null;
    setIsSimulating(false);
    setSimulationError('');
    setSimulationResult(null);
  };

  const onTeamChange = (event) => {
    workspaceVersion.current += 1;
    cancelSimulation();
    setPlayers([]);
    setRosterError('');
    setSelectedTeam(event.target.value);
    setLineup(Array(9).fill(null));
    setRosterReload(value => value + 1);
  };

  const movePlayerToSlot = (player, index) => {
    cancelSimulation();

    const newLineup = [...lineup];
    const existingPlayer = newLineup[index];

    // the player is already in the lineup
    const playerIndex = lineup.indexOf(player);
    if (playerIndex !== -1) {
      newLineup[index] = player;
      newLineup[playerIndex] = existingPlayer;
      setLineup(newLineup);
      return;
    }

    // move the existing player back to the roster if there is one
    if (existingPlayer) {
      setPlayers(prevPlayers => 
        [...prevPlayers, existingPlayer].sort((a, b) => a.name.localeCompare(b.name))
      );
    }

    newLineup[index] = player;
    setLineup(newLineup);
    setPlayers(prevPlayers => 
      prevPlayers.filter(p => p.name !== player.name)
    );
  };

  const removePlayerFromSlot = (index) => {
    cancelSimulation();
    const player = lineup[index];
    const newLineup = [...lineup];
    newLineup[index] = null;
    setLineup(newLineup);
    setPlayers([...players, player].sort((a, b) => a.name.localeCompare(b.name)));
  };

  const simulateLineup = async () => {
    if (simulationRequest.current || lineup.some(player => !player)) return;
    const controller = new AbortController();
    simulationRequest.current = controller;
    const version = simulationVersion.current;
    const workspace = workspaceVersion.current;
    setIsSimulating(true);
    setSimulationError('');
    setSimulationResult(null);
    const separator = selectedTeam.lastIndexOf('+');
    const payload = {
      team: selectedTeam.slice(0, separator),
      year: Number(selectedTeam.slice(separator + 1)),
      batter_ids: lineup.map(player => player.id),
    };
    try {
      const response = await fetch(`${apiBasePath}/simulations`, {
        method: 'POST',
        signal: controller.signal,
        credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': readCookie(csrfCookieName) || '' },
        body: JSON.stringify(payload),
      });
      if (controller.signal.aborted || workspace !== workspaceVersion.current || version !== simulationVersion.current) return;
      if (response.status === 401) { leaveExpiredSession(); return; }
      if (!response.ok) throw new Error(`Simulation failed. ${await parseAPIError(response)}`);
      const result = await response.json();
      if (!Number.isFinite(result.average_score) || !Number.isFinite(result.average_hits)) throw new Error('The simulation response could not be read.');
      if (!controller.signal.aborted && workspace === workspaceVersion.current && version === simulationVersion.current) {
        if (Array.isArray(result.batters) && result.batters.length === 9) setLineup(result.batters.map(decoratePlayer));
        setSimulationResult(result);
        setHistoryRefresh(value => value + 1);
      }
    } catch (error) {
      if (!controller.signal.aborted && workspace === workspaceVersion.current && version === simulationVersion.current) setSimulationError(error.message || 'Simulation failed. Please try again.');
    } finally {
      if (simulationRequest.current === controller) {
        simulationRequest.current = null;
        setIsSimulating(false);
      }
    }
  };

  const leaveExpiredSession = () => {
    setCurrentUser(null);
    setAuthStatus('anonymous');
    setAuthMessage('Your session expired. Please log in again.');
    resetWorkspace();
  };

  const loadSavedLineup = saved => {
    if (!Array.isArray(saved.batters) || saved.batters.length !== 9) throw new Error('The saved lineup could not be read.');
    workspaceVersion.current += 1;
    cancelSimulation();
    const teamKey = `${saved.team}+${saved.year}`;
    setTeams(previous => previous.some(team => team.name === saved.team && team.year === saved.year)
      ? previous : [...previous, { name: saved.team, year: saved.year }]);
    setPlayers([]);
    setSelectedTeam(teamKey);
    setLineup(saved.batters.map(decoratePlayer));
    setRosterReload(value => value + 1);
  };

  const onAuthFormChange = (event) => {
    const { name, value } = event.target;
    setAuthForm(prevForm => ({
      ...prevForm,
      [name]: value,
    }));
  };

  const resetWorkspace = () => {
    workspaceVersion.current += 1;
    cancelSimulation();
    setRosterError('');
    setRosterLoading(false);
    setTeamsLoading(false);
    setTeamsError('');
    setTeams([]);
    setSelectedTeam('');
    setPlayers([]);
    setLineup(Array(9).fill(null));
    setSimulationResult(null);
  };

  const loginWithCredentials = async (usernameOrEmail, password) => {
    const response = await fetch(`${apiBasePath}/auth/login`, {
      method: 'POST',
      credentials: 'same-origin',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        username_or_email: usernameOrEmail,
        password,
      }),
    });

    if (!response.ok) {
      throw new Error(await parseAPIError(response));
    }

    return response.json();
  };

  const submitAuthForm = async (event) => {
    event.preventDefault();
    if (isSubmitting) {
      return;
    }

    setIsSubmitting(true);
    setAuthError('');
    setAuthMessage('');

    try {
      let user;
      if (authMode === 'register') {
        const response = await fetch(`${apiBasePath}/auth/register`, {
          method: 'POST',
          credentials: 'same-origin',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            username: authForm.username,
            email: authForm.email,
            password: authForm.password,
          }),
        });

        if (!response.ok) {
          throw new Error(await parseAPIError(response));
        }

        user = await loginWithCredentials(authForm.username, authForm.password);
      } else {
        user = await loginWithCredentials(authForm.username, authForm.password);
      }

      setCurrentUser(user);
      setAuthStatus('authenticated');
      setAuthMessage('');
      setAuthForm({
        username: '',
        email: '',
        password: '',
      });
    } catch (error) {
      setAuthError(error.message);
    } finally {
      setIsSubmitting(false);
    }
  };

  const logout = async () => {
    if (logoutPending.current) return;
    logoutPending.current = true;
    setIsLoggingOut(true);
    setAuthError('');
    setAuthMessage('');

    try {
      const csrfToken = readCookie(csrfCookieName);
      const response = await fetch(`${apiBasePath}/auth/logout`, {
        method: 'POST',
        credentials: 'same-origin',
        headers: {
          'X-CSRF-Token': csrfToken || '',
        },
      });

      let sessionExpired = response.status === 401;
      if (response.status === 403) {
        const sessionResponse = await fetch(`${apiBasePath}/users/me`, {
          credentials: 'same-origin',
        });
        sessionExpired = sessionResponse.status === 401;
      }
      if (!response.ok && !sessionExpired) {
        setAuthError(await parseAPIError(response));
        return;
      }

      setCurrentUser(null);
      setAuthStatus('anonymous');
      setAuthMessage('You have been logged out.');
      resetWorkspace();
    } catch {
      setAuthError('Network error. Please try again.');
    } finally {
      logoutPending.current = false;
      setIsLoggingOut(false);
    }
  };

  const switchAuthMode = (mode) => {
    setAuthMode(mode);
    setAuthError('');
    setAuthMessage('');
  };

  return (
    <DndProvider backend={HTML5Backend}>
      <header className="app-header">
        <h1>Lineup Lab</h1>
        {currentUser && (
          <div className="user-menu" aria-label="User session">
            <span>Signed in as <strong>{currentUser.username}</strong></span>
            <button type="button" className="secondary-button" onClick={logout} disabled={isLoggingOut}>{isLoggingOut ? 'Logging out...' : 'Log out'}</button>
          </div>
        )}
      </header>

      {authStatus === 'loading' && (
        <p className="session-status">Checking session...</p>
      )}

      {authStatus === 'authenticated' && authError && (
        <p className="auth-error" role="alert">{authError}</p>
      )}

      {authStatus === 'anonymous' && (
        <div className="auth-layout">
          <form className="auth-card" onSubmit={submitAuthForm}>
            <div className="auth-tabs" role="tablist" aria-label="Authentication mode">
              <button
                type="button"
                role="tab"
                aria-selected={authMode === 'login'}
                className={authMode === 'login' ? 'active' : ''}
                onClick={() => switchAuthMode('login')}
              >
                Log in
              </button>
              <button
                type="button"
                role="tab"
                aria-selected={authMode === 'register'}
                className={authMode === 'register' ? 'active' : ''}
                onClick={() => switchAuthMode('register')}
              >
                Register
              </button>
            </div>
            <label>
              {authMode === 'login' ? 'Username or email' : 'Username'}
              <input
                name="username"
                value={authForm.username}
                onChange={onAuthFormChange}
                autoComplete={authMode === 'login' ? 'username' : 'username'}
                required
              />
            </label>
            {authMode === 'register' && (
              <label>
                Email
                <input
                  type="email"
                  name="email"
                  value={authForm.email}
                  onChange={onAuthFormChange}
                  autoComplete="email"
                  required
                />
              </label>
            )}
            <label>
              Password
              <input
                type="password"
                name="password"
                value={authForm.password}
                onChange={onAuthFormChange}
                autoComplete={authMode === 'login' ? 'current-password' : 'new-password'}
                required
              />
            </label>
            {authError && <p className="auth-error" role="alert">{authError}</p>}
            {authMessage && <p className="auth-message">{authMessage}</p>}
            <button type="submit" className="primary-button" disabled={isSubmitting}>
              {isSubmitting ? 'Submitting...' : (authMode === 'login' ? 'Log in' : 'Create account')}
            </button>
          </form>
        </div>
      )}

      {authStatus === 'authenticated' && (
        <>
        <div className="container">
          <div className="card">
            <Lineup
              lineup={lineup}
              movePlayerToSlot={movePlayerToSlot}
              removePlayerFromSlot={removePlayerFromSlot}
              simulateLineup={simulateLineup}
              simulationResult={simulationResult}
              canSimulate={Boolean(selectedTeam) && lineup.every(Boolean)}
              isSimulating={isSimulating}
              simulationError={simulationError}
            />
          </div>
          <div className="card">
            {teamsLoading && <p role="status">Loading teams...</p>}
            {teamsError && <p role="alert">{teamsError} <button type="button" onClick={() => setTeamsReload(value => value + 1)}>Retry teams</button></p>}
            {rosterLoading && <p role="status">Loading roster...</p>}
            {rosterError && <p role="alert">{rosterError} <button type="button" onClick={() => setRosterReload(value => value + 1)}>Retry roster</button></p>}
            <Roster players={players} teams={teams} selectedTeam={selectedTeam} onTeamChange={onTeamChange} />
          </div>
        </div>
        <div className="container">
          <SavedLineups key={`saved-${currentUser.id}`} selectedTeam={selectedTeam} lineup={lineup} onLoad={loadSavedLineup} onSessionExpired={leaveExpiredSession} />
        </div>
        <PersonalHistory key={`history-${currentUser.id}`} refreshToken={historyRefresh} onSessionExpired={leaveExpiredSession} />
        </>
      )}
    </DndProvider>
  );
};

export default App;
