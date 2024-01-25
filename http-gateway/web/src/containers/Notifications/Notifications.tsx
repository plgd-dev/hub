import { useIntl } from 'react-intl'

import { messages as menuT } from '@shared-ui/components/Atomic/Menu/Menu.i18n'
import PageLayout from '@shared-ui/components/Atomic/PageLayout'

const Notifications = () => {
    const { formatMessage: _ } = useIntl()

    return (
        <PageLayout title={_(menuT.notifications)}>
            <div />
        </PageLayout>
    )
}

export default Notifications
