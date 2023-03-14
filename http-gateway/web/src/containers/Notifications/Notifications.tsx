import { useIntl } from 'react-intl'
import { messages as menuT } from '@shared-ui/components/new/Menu/Menu.i18n'
import PageLayout from '@shared-ui/components/new/PageLayout'

const Notifications = () => {
    const { formatMessage: _ } = useIntl()

    return (
        <PageLayout
            breadcrumbs={[
                {
                    to: '/',
                    label: _(menuT.dashboard),
                },
                {
                    label: _(menuT.notifications),
                },
            ]}
            title={_(menuT.notifications)}
        >
            <div />
        </PageLayout>
    )
}

export default Notifications
