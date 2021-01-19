import { memo } from 'react'

import logoBig from '@/assets/img/logo-big.svg'
import logoSmall from '@/assets/img/logo-small.svg'
import './left-panel.scss'

export const LeftPanel = memo(({ children }) => {
  return (
    <div id="left-panel">
      <a href="/" className="logo">
        <img src={logoBig} alt="PLGD Logo Big" className="logo-big" />
        <img src={logoSmall} alt="PLGD Logo Small" className="logo-small" />
      </a>
      {children}
    </div>
  )
})
