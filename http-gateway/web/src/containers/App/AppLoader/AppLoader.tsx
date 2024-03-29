import { useIntl } from 'react-intl'

import PageLoader from '@shared-ui/components/Atomic/PageLoader'

import { messages as g } from '../../Global.i18n'

const AppLoader = () => {
    const { formatMessage: _ } = useIntl()

    return (
        <>
            <PageLoader loading noOffset className='auth-loader' />
            <div className='page-loading-text'>{`${_(g.loading)}...`}</div>
        </>
    )
}

AppLoader.displayName = 'AppLoader'

export default AppLoader
