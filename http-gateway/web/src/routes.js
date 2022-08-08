import { Switch, Route } from 'react-router-dom'

import { DevicesListPage, DevicesDetailsPage } from '@/containers/devices'
import { PendingCommandsListPage } from '@/containers/pending-commands'
import { Notifications } from '@/containers/notifications'
import { Configuration } from '@/containers/configuration'
import { NotFoundPage } from '@/containers/not-found-page'

export const Routes = () => (
  <Switch>
    <Route exact path="/" component={DevicesListPage} />
    <Route
      path={['/devices/:id', '/devices/:id/:href*']}
      component={DevicesDetailsPage}
    />
    <Route path="/pending-commands" component={PendingCommandsListPage} />
    <Route path="/notifications">
      <Notifications />
    </Route>
    <Route path="/configuration">
      <Configuration />
    </Route>
    <Route path="*">
      <NotFoundPage />
    </Route>
  </Switch>
)
