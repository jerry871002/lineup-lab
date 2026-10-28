import React, { useEffect, useRef, useState } from 'react';
import './SavedLineups.css';

const csrfToken = () => {
  const entry = document.cookie.split(/;\s*/).find(value => value.startsWith('lineup_lab_csrf='));
  const raw = entry ? entry.slice('lineup_lab_csrf='.length) : '';
  const value = raw.startsWith('"') && raw.endsWith('"') ? raw.slice(1, -1) : raw;
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
};

const teamSeason = (selectedTeam) => {
  const separator = selectedTeam.lastIndexOf('+');
  const team = selectedTeam.slice(0, separator);
  const year = Number(selectedTeam.slice(separator + 1));
  return separator > 0 && team.trim() && Number.isInteger(year) && year >= 1900 && year <= 2100
    ? { team, year } : null;
};

const validLineup = (lineup) => {
  if (lineup.length !== 9 || lineup.some(player => !player)) return false;
  const fields = ['at_bat', 'hit', 'double', 'triple', 'home_run', 'ball_on_base', 'hit_by_pitch'];
  return lineup.every(player => Number.isSafeInteger(player.id) && player.id > 0
    && typeof player.name === 'string' && player.name.trim()
    && fields.every(field => Number.isSafeInteger(player[field]) && player[field] >= 0)
    && player.at_bat > 0 && player.hit <= player.at_bat
    && player.double + player.triple + player.home_run <= player.hit)
    && new Set(lineup.map(player => player.id)).size === 9
    && new Set(lineup.map(player => player.name.trim().toLowerCase())).size === 9
    && lineup.some(player => player.at_bat > player.hit);
};

const request = async (path, signal, method = 'GET', body) => {
  const response = await fetch(`/api/lineups${path}`, {
    method,
    credentials: 'same-origin',
    signal,
    ...(method !== 'GET' ? { headers: {
      'X-CSRF-Token': csrfToken(),
      ...(body ? { 'Content-Type': 'application/json' } : {}),
    } } : {}),
    ...(body ? { body: JSON.stringify(body) } : {}),
  });
  let data = null;
  if (response.status !== 204) {
    try {
      data = await response.json();
    } catch {
      if (response.ok) throw new Error('The saved lineup response could not be read.');
    }
  }
  if (signal.aborted) throw new DOMException('Request cancelled', 'AbortError');
  if (!response.ok) {
    const error = new Error(typeof data?.detail === 'string' ? data.detail : 'Saved lineup request failed. Please try again.');
    error.status = response.status;
    throw error;
  }
  return data;
};

const SavedLineups = ({ selectedTeam, lineup, onLoad, onSessionExpired }) => {
  const [items, setItems] = useState([]);
  const [selectedId, setSelectedId] = useState('');
  const [current, setCurrent] = useState(null);
  const [name, setName] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [message, setMessage] = useState('');
  const [conflict, setConflict] = useState(null);
  const active = useRef(null);
  const sessionExpired = useRef(onSessionExpired);
  sessionExpired.current = onSessionExpired;

  const run = async (action, conflictId = '') => {
    if (active.current) return;
    const controller = new AbortController();
    active.current = controller;
    setBusy(true);
    setError('');
    setMessage('');
    setConflict(null);
    try {
      await action(controller.signal);
    } catch (failure) {
      if (controller.signal.aborted) return;
      if (failure.status === 401) {
        setItems([]);
        setSelectedId('');
        setCurrent(null);
        setError('Your session expired. Please log in again.');
        sessionExpired.current?.();
      } else {
        setError(failure.message || 'Network error. Please try again.');
        if (failure.status === 409) setConflict({ id: conflictId });
      }
    } finally {
      if (active.current === controller) {
        active.current = null;
        setBusy(false);
      }
    }
  };

  const fetchList = async (signal) => {
    const data = await request('', signal);
    if (!Array.isArray(data)) throw new Error('The saved lineup list could not be read.');
    setItems(data);
  };

  useEffect(() => {
    run(fetchList);
    return () => {
      active.current?.abort();
      active.current = null;
    };
  }, []);

  const remember = (item) => {
    setItems(previous => [item, ...previous.filter(value => value.id !== item.id)].slice(0, 100));
    setSelectedId(String(item.id));
    setCurrent(item);
    setName(item.name);
  };

  const load = (id) => run(async signal => {
    const item = await request(`/${id}`, signal);
    remember(item);
    onLoad(item);
    setMessage(`Loaded ${item.name}.`);
  });

  const season = teamSeason(selectedTeam);
  const canSave = Boolean(season && validLineup(lineup) && name.trim() && [...name.trim()].length <= 100);
  const save = (update) => {
    if (!canSave || (update && !current)) return;
    const target = update ? current : null;
    run(async signal => {
      const item = await request(target ? `/${target.id}` : '', signal, target ? 'PUT' : 'POST', {
        name: name.trim(),
        ...season,
        batter_ids: lineup.map(player => player.id),
        ...(target ? { revision: target.revision } : {}),
      });
      remember(item);
      setMessage(`Saved ${item.name}.`);
    }, target ? String(target.id) : '');
  };

  const remove = () => {
    if (!selectedId) return;
    const id = selectedId;
    run(async signal => {
      await request(`/${id}`, signal, 'DELETE');
      setItems(previous => previous.filter(item => String(item.id) !== id));
      setSelectedId('');
      if (String(current?.id) === id) {
        setCurrent(null);
        setName('');
      }
      setMessage('Saved lineup deleted.');
    });
  };

  return (
    <section className="saved-lineups" aria-label="Saved lineups" aria-busy={busy}>
      <h2>Saved lineups</h2>
      <p>Your latest 100 saved lineups. Loading replaces the current batting order.</p>
      <label>
        Saved lineup
        <select value={selectedId} onChange={event => setSelectedId(event.target.value)} disabled={busy}>
          <option value="">Select a saved lineup</option>
          {items.map(item => <option key={item.id} value={item.id}>{item.name} — {item.team} ({item.year})</option>)}
        </select>
      </label>
      {!busy && items.length === 0 && <p>No saved lineups yet.</p>}
      <div className="saved-lineups-actions">
        <button type="button" onClick={() => load(selectedId)} disabled={busy || !selectedId}>Load selected lineup</button>
        <button type="button" onClick={remove} disabled={busy || !selectedId}>Delete selected lineup</button>
        <button type="button" onClick={() => run(fetchList)} disabled={busy}>Refresh saved lineups</button>
      </div>
      <form onSubmit={event => { event.preventDefault(); save(false); }}>
        <label>
          Lineup name
          <input value={name} onChange={event => setName(event.target.value)} maxLength={100} required disabled={busy} />
        </label>
        <p>Choose a team-season and fill all nine slots with distinct players before saving.</p>
        {current && <p>Updating: <strong>{current.name}</strong> ({current.team}, {current.year})</p>}
        <div className="saved-lineups-actions">
          <button type="submit" disabled={busy || !canSave}>Save as new lineup</button>
          <button type="button" onClick={() => save(true)} disabled={busy || !canSave || !current}>Update loaded lineup</button>
        </div>
      </form>
      {busy && <p role="status">Working on saved lineups...</p>}
      {message && <p role="status">{message}</p>}
      {error && <p role="alert">{error}</p>}
      {conflict && <button type="button" disabled={busy} onClick={() => conflict.id ? load(conflict.id) : run(fetchList)}>
        {conflict.id ? 'Reload saved lineup' : 'Reload saved lineups'}
      </button>}
    </section>
  );
};

export default SavedLineups;
