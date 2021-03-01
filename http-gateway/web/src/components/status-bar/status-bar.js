import { memo } from 'react'

import { LanguageSwitcher } from '@/components/language-switcher'
import { UserWidget } from '@/components/user-widget'

import './status-bar.scss'

export const StatusBar = memo(() => {
  return (
    <>
      <div id="status-bar-shadow" className="status-bar" />
      <header id="status-bar" className="status-bar">
        {/* Insert custom components here. */}
        <LanguageSwitcher />
        <UserWidget />
      </header>
    </>
  )
})
