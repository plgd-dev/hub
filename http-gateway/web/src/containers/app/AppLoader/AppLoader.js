import { PageLoader } from '@/components/page-loader'
import { messages as t } from '../app-i18n'
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
