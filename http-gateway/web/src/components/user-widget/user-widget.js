import { useRef, useState, memo } from 'react'
import classNames from 'classnames'
import { useIntl } from 'react-intl'

import { useClickOutside } from '@/common/hooks'
import { messages as t } from './user-widget-i18n'
import './user-widget.scss'
import { useAuth } from 'oidc-react'

export const UserWidget = memo(() => {
  const [expanded, setExpand] = useState(false)
  const { isLoading, userData, logout } = useAuth()
  const { formatMessage: _ } = useIntl()
  const clickRef = useRef()
  useClickOutside(clickRef, () => setExpand(false))

  if (!isLoading && !userData) {
    return null
  }

  return (
    <div id="user-widget" className="status-bar-item" ref={clickRef}>
      <div className="toggle" onClick={() => setExpand(!expanded)}>
        {/*<div className="user-avatar">*/}
        {/*<img src={user.picture} alt="User Avatar" />*/}
        {/*</div>*/}
        {userData.profile.name}
        <i
          className={classNames('fas', {
            'fa-chevron-down': !expanded,
            'fa-chevron-up': expanded,
          })}
        />
      </div>
      {expanded && (
        <div className="content">
          <span onClick={() => logout({ returnTo: window.location.origin })}>
            {_(t.logOut)}
          </span>
        </div>
      )}
    </div>
  )
})
