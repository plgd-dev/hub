import { useContext, useRef, useState, memo } from 'react'
import classNames from 'classnames'

import { useClickOutside } from '@/common/hooks'
import { LanguageContext } from '@/components/intl-provider'
import './language-switcher.scss'

export const LanguageSwitcher = memo(() => {
  const [expanded, setExpand] = useState(false)
  const clickRef = useRef()
  useClickOutside(clickRef, () => setExpand(false))
  const { language, setLanguage } = useContext(LanguageContext)

  const changeLanguage = lang => {
    setLanguage(lang)
    setExpand(false)
  }

  return (
    <div id="language-switcher" className="status-bar-item" ref={clickRef}>
      <div className="toggle" onClick={() => setExpand(!expanded)}>
        {language}
        <i className={classNames('fas', { 'fa-chevron-down': !expanded, 'fa-chevron-up': expanded })} />
      </div>
      {expanded && (
        <div className="content">
          <span onClick={() => changeLanguage('en')}>English</span>
          <span onClick={() => changeLanguage('de')}>Deutsch</span>
        </div>
      )}
    </div>
  )
})
