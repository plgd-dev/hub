import { FC } from 'react'
import { useIntl } from 'react-intl'
import { messages as t } from './NotFoundPage.i18n'
import { Props } from './NotFoundPage.types'
import PageLayout from '@shared-ui/components/new/PageLayout'

const NotFoundPage: FC<Props> = ({ title, message }) => {
    const { formatMessage: _ } = useIntl()
    const pageTitle = title || _(t.pageTitle)

    return (
        <PageLayout title={pageTitle}>
            <h2>{pageTitle}</h2>
            {message || _(t.notFoundPageDefaultMessage)}
        </PageLayout>
    )
}

NotFoundPage.displayName = 'NotFoundPage'

export default NotFoundPage
