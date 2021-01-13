import { memo } from 'react'
import { useIntl } from 'react-intl'

import { messages as t } from './footer-i18n'
import './footer.scss'

export const Footer = memo(() => {
  const { formatMessage: _ } = useIntl()

  return (
    <footer id="footer">
      <a href="/" target="_blank">
        {_(t.API)}
      </a>
      <a href="/" target="_blank">
        {_(t.docs)}
      </a>
      <a href="/" target="_blank">
        {_(t.contribute)}
      </a>
    </footer>
  )
})
