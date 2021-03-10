import { Switch, Route } from 'react-router-dom'

import { Dashboard } from '@/containers/dashboard'
import { ThingsListPage, ThingsDetailsPage } from '@/containers/things'
import { Notifications } from '@/containers/notifications'
import { Configuration } from '@/containers/configuration'
import { NotFoundPage } from '@/containers/not-found-page'

export const Routes = () => {
  return (
    <Switch>
      <Route exact path="/">
        <Dashboard />
      </Route>
      <Route path="/things" exact>
        <ThingsListPage />
      </Route>
      <Route path={['/things/:id', '/things/:id/:href*']} exact>
        <ThingsDetailsPage />
      </Route>
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
}
