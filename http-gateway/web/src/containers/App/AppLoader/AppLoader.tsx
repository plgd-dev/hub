import PageLoader from '@shared-ui/components/new/PageLoader'
import { messages as t } from '../App.i18n'
import { useIntl } from 'react-intl'

const AppLoader = () => {
  const { formatMessage: _ } = useIntl()

  return (
    <>
      <PageLoader className="auth-loader" loading />
      <div className="page-loading-text">{`${_(t.loading)}...`}</div>
    </>
  )
}

AppLoader.displayName = 'AppLoader'

export default AppLoader
