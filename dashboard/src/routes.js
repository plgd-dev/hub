import { Switch, Route } from 'react-router-dom'

import { Dashboard } from '@/containers/dashboard'
import { Things } from '@/containers/things'
import { Services } from '@/containers/services'
import { Notifications } from '@/containers/notifications'

export const Routes = () => {
  return (
    <Switch>
      <Route path="/things">
        <Things />
      </Route>
      <Route path="/services">
        <Services />
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
