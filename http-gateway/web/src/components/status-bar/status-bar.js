import { memo } from 'react'
import classNames from 'classnames'

import { LanguageSwitcher } from '@/components/language-switcher'
import { UserWidget } from '@/components/user-widget'

import './status-bar.scss'

export const StatusBar = memo(({ collapsed }) => {
  return (
    <div id="status-bar" className={classNames({ collapsed })}>
      <LanguageSwitcher />
      <UserWidget />
    </div>
  )
})
