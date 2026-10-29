# SavedLineups integration

Import `SavedLineups` from `./SavedLineups.jsx` and mount it only inside the
session-authenticated workspace. Key it by the authenticated user ID so a user
change unmounts the previous user's component and aborts any pending request:

```jsx
<SavedLineups
  key={currentUser.id}
  selectedTeam={selectedTeam}
  lineup={lineup}
  onLoad={loadSavedLineup}
  onSessionExpired={leaveExpiredSession}
/>
```

- `selectedTeam` uses the existing `team+year` format.
- `lineup` is the nine ordered player objects with positive numeric `id` and
  canonical batting counts. Empty/invalid/duplicate lineups cannot be saved.
- `onLoad(savedItem)` receives the latest complete server response, including
  its ordered `batters` snapshot. Set the workspace team-season, derive AVG/OBP/SLG
  for display, rebuild the roster without the loaded IDs, and clear simulation
  results. Preserve the saved snapshot rather than replacing its counts with
  current roster data. Team/roster fetch coordination belongs to the App adapter.
- `onSessionExpired()` must leave the authenticated workspace and clear its
  private data. Both list reads and writes returning 401 invoke it.

Load always fetches `/api/lineups/{id}`. Update targets the last loaded, created,
or updated item using its observed revision; merely selecting another entry
only changes the Load/Delete selection. A 409 exposes an explicit reload button
instead of silently replacing edits. All requests are serialized, use the same
origin, and abort on unmount. Writes send the readable CSRF cookie after handling
the auth service's quoted cookie format.

The App adapter now mounts this panel inside the authenticated workspace, restores
snapshot display statistics, refreshes the remaining roster, and clears obsolete
results. `onSimulate(savedItem, signal)` computes the latest server-saved revision
through the protected history endpoint; `simulationBusy` prevents concurrent runs.
PersonalHistory refreshes after a recorded run and both panels remount per user.
