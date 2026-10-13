import { render } from '@testing-library/react';
import { MemoryRouter, useLocation } from 'react-router-dom';
import { App } from '../App';
import { AuthProvider } from '../auth/AuthContext';
import { ROUTER_FUTURE } from '../routerFuture';

/** Exposes the current URL so tests can assert on redirects. */
function LocationProbe() {
  const location = useLocation();
  return <output data-testid="location">{location.pathname + location.search}</output>;
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
