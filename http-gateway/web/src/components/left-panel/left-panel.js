import { memo } from 'react'
import { Link } from 'react-router-dom'

import logoBig from '@/assets/img/plgd-logo-full-white.svg'
import logoSmall from '@/assets/img/plgd-logo-alt-white.svg'
import './left-panel.scss'

export const LeftPanel = memo(({ children }) => {
  return (
    <div id="left-panel">
      <Link to="/" className="logo">
        <img src={logoBig} alt="PLGD Logo Big" className="logo-big" />
        <img src={logoSmall} alt="PLGD Logo Small" className="logo-small" />
      </Link>
      {children}
    </div>
  )
})
