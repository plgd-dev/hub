import { useRef, useState, memo } from 'react'
import classNames from 'classnames'
import { useIntl } from 'react-intl'
import Avatar from 'react-avatar'
import { useClickOutside } from '@/common/hooks'
import { messages as t } from './user-widget-i18n'
import './user-widget.scss'
import { useAuth } from 'oidc-react'

export const UserWidget = memo(() => {
  const [expanded, setExpand] = useState(false)
  const { isLoading, userData, signOutRedirect } = useAuth()
  const { formatMessage: _ } = useIntl()
  const clickRef = useRef()
  useClickOutside(clickRef, () => setExpand(false))

  if (!isLoading && !userData) {
    return null
  }

  return (
    <div id="user-widget" className="status-bar-item" ref={clickRef}>
      <div className="toggle" onClick={() => setExpand(!expanded)}>
        <div className="user-avatar">
          <Avatar
            name={userData.profile.name}
            size="30"
            round="50%"
            color="#255897"
          />
        </div>
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
          <span onClick={() => signOutRedirect()}>{_(t.logOut)}</span>
        </div>
      )}
    </div>
  )
})
