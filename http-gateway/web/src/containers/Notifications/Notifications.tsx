import { useIntl } from 'react-intl'
import Layout from '@shared-ui/components/new/Layout'
import { messages as menuT } from '@shared-ui/components/new/Menu/Menu.i18n'

const Notifications = () => {
  const { formatMessage: _ } = useIntl()

  return (
    <Layout
      title={_(menuT.notifications)}
      breadcrumbs={[
        {
          to: '/',
          label: _(menuT.dashboard),
        },
        {
          label: _(menuT.notifications),
        },
      ]}
    >
      <div />
    </Layout>
  )
}

export default Notifications
