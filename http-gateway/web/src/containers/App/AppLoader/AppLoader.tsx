import { useIntl } from 'react-intl'

import FullPageLoader from '@shared-ui/components/Atomic/FullPageLoader'

import { messages as g } from '../../Global.i18n'

const AppLoader = () => {
    const { formatMessage: _ } = useIntl()

    return <FullPageLoader i18n={{ loading: _(g.loading) }} />
}

AppLoader.displayName = 'AppLoader'

export default AppLoader
