/* eslint-disable react/jsx-no-target-blank */
import { memo } from 'react'
import { useIntl } from 'react-intl'

import { messages as t } from './footer-i18n'
import './footer.scss'

export const Footer = memo(() => {
  const { formatMessage: _ } = useIntl()

  return (
    <footer id="footer">
      <div className="left" />
      <div className="right">
        <a
          href="https://github.com/plgd-dev/cloud/raw/master/http-gateway/swagger.yaml"
          target="_blank"
          rel="noopener"
        >
          {_(t.API)}
        </a>
        <a href="https://plgd.dev/documentation" target="_blank" rel="noopener">
          {_(t.docs)}
        </a>
        <a
          href="https://github.com/plgd-dev/cloud"
          target="_blank"
          rel="noopener"
        >
          {_(t.contribute)}
        </a>
      </div>
    </footer>
  )
})
