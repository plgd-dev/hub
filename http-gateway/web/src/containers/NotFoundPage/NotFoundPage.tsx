import { FC } from 'react'
import { useIntl } from 'react-intl'
import Layout from '@shared-ui/components/new/Layout'
import { messages as t } from './NotFoundPage.i18n'
import { Props } from './NotFoundPage.types'

const NotFoundPage: FC<Props> = ({ title, message }) => {
  const { formatMessage: _ } = useIntl()
  const pageTitle = title || _(t.pageTitle)

  return (
    <Layout title={pageTitle}>
      <h2>{pageTitle}</h2>
      {message || _(t.notFoundPageDefaultMessage)}
    </Layout>
  )
}

NotFoundPage.displayName = 'NotFoundPage'

export default NotFoundPage
