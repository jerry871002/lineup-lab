import React, { useState, useEffect } from 'react';
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import Lineup from './Lineup.jsx';
import Roster from './Roster.jsx';
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

  const value = cookie.slice(prefix.length);
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
  const [teams, setTeams] = useState([]);
  const [selectedTeam, setSelectedTeam] = useState('');
  const [players, setPlayers] = useState([]);
  const [lineup, setLineup] = useState(Array(9).fill(null));
  const [simulationResult, setSimulationResult] = useState(null);

  useEffect(() => {
    fetch(`${apiBasePath}/users/me`, {
      credentials: 'same-origin',
    })
      .then(async response => {
        if (response.ok) {
          const user = await response.json();
          setCurrentUser(user);
          setAuthStatus('authenticated');
          return;
        }

        setCurrentUser(null);
        setAuthStatus('anonymous');
      })
      .catch(error => {
        console.error('Error checking session:', error);
        setCurrentUser(null);
        setAuthStatus('anonymous');
      });
  }, []);

  useEffect(() => {
    if (authStatus !== 'authenticated') {
      return;
    }

    fetch(`${apiBasePath}/teams`)
      .then(response => {
        if (!response.ok) {
          throw new Error('Failed to fetch teams');
        }

        return response.json();
      })
      .then(data => setTeams(Array.isArray(data) ? data : []))
      .catch(error => console.error('Error fetching teams:', error));
  }, [authStatus]);

  useEffect(() => {
    if (selectedTeam) {
      const [name, year] = selectedTeam.split('+');
      fetch(`${apiBasePath}/batting?team=${name}&year=${year}`)
        .then(response => {
          if (!response.ok) {
            throw new Error('Failed to fetch roster');
          }

          return response.json();
        })
        .then(data => {
          if (!Array.isArray(data)) {
            setPlayers([]);
            return;
          }

          // add AVG, OBP, SLG to the player data
          data.map(player => {
            player.avg = player.hit / player.at_bat;
            player.obp = (player.hit + player.ball_on_base + player.hit_by_pitch) / (player.at_bat + player.ball_on_base + player.hit_by_pitch);
            player.slg = (player.hit + player.double + 2 * player.triple + 3 * player.home_run) / player.at_bat;
            return player;
          });
          setPlayers(data.sort((a, b) => a.name.localeCompare(b.name)));
        })
        .catch(error => console.error('Error fetching players:', error));
    }
  }, [selectedTeam]);

  const onTeamChange = (event) => {
    setSelectedTeam(event.target.value);
    setLineup(Array(9).fill(null));
  };

  const movePlayerToSlot = (player, index) => {
    setSimulationResult(null);

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
    setSimulationResult(null);
    const player = lineup[index];
    const newLineup = [...lineup];
    newLineup[index] = null;
    setLineup(newLineup);
    setPlayers([...players, player].sort((a, b) => a.name.localeCompare(b.name)));
  };

  const simulateLineup = async () => {
    const response = await fetch(`${apiBasePath}/simulate`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(lineup),
    });
    const result = await response.json();
    console.log(result);
    setSimulationResult(result);
  };

  const onAuthFormChange = (event) => {
    const { name, value } = event.target;
    setAuthForm(prevForm => ({
      ...prevForm,
      [name]: value,
    }));
  };

  const resetWorkspace = () => {
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

      if (!response.ok) {
        setAuthError(await parseAPIError(response));
        return;
      }

      setCurrentUser(null);
      setAuthStatus('anonymous');
      setAuthMessage('You have been logged out.');
      resetWorkspace();
    } catch {
      setAuthError('Network error. Please try again.');
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
            <button type="button" className="secondary-button" onClick={logout}>Log out</button>
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
        <div className="container">
          <div className="card">
            <Lineup
              lineup={lineup}
              movePlayerToSlot={movePlayerToSlot}
              removePlayerFromSlot={removePlayerFromSlot}
              simulateLineup={simulateLineup}
              simulationResult={simulationResult}
            />
          </div>
          <div className="card">
            <Roster players={players} teams={teams} selectedTeam={selectedTeam} onTeamChange={onTeamChange} />
          </div>
        </div>
      )}
    </DndProvider>
  );
};

export default App;
