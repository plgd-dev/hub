import { useIntl } from 'react-intl'

import { Layout } from '@shared-ui/components/old/layout'

import { messages as t } from './not-found-page-i18n'

export const NotFoundPage = ({ title, message }) => {
  const { formatMessage: _ } = useIntl()
  const pageTitle = title || _(t.pageTitle)

  return (
    <Layout title={pageTitle}>
      <h2>{pageTitle}</h2>
      {message || _(t.notFoundPageDefaultMessage)}
    </Layout>
  )
}
