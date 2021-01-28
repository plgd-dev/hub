import { Switch, Route } from 'react-router-dom'

import { Dashboard } from '@/containers/dashboard'
import { ThingsListPage, ThingsDetailsPage } from '@/containers/things'
import { Services } from '@/containers/services'
import { Notifications } from '@/containers/notifications'

export const Routes = () => {
  return (
    <Switch>
      <Route path="/things" exact>
        <ThingsListPage />
      </Route>
      <Route path="/things/:id">
        <ThingsDetailsPage />
      </Route>
      <Route path="/notifications">
        <Notifications />
      </Route>
      <Route path="/">
        <Dashboard />
      </Route>
    </Switch>
  )
}
