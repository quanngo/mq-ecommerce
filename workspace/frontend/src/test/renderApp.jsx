import { render } from '@testing-library/react';
import { MemoryRouter, useLocation } from 'react-router-dom';
import { App } from '../App';
import { AuthProvider } from '../auth/AuthContext';
import { ROUTER_FUTURE } from '../routerFuture';

/** Exposes the current URL so tests can assert on redirects. */
function LocationProbe() {
  const location = useLocation();
  // A plain div: <output> has the implicit role "status" and would clash with
  // the pages' own role="status" messages in findByRole('status').
  return <div data-testid="location">{location.pathname + location.search}</div>;
}

export function renderApp(initialPath) {
  return render(
    <MemoryRouter initialEntries={[initialPath]} future={ROUTER_FUTURE}>
      <AuthProvider>
        <App />
        <LocationProbe />
      </AuthProvider>
    </MemoryRouter>,
  );
}
