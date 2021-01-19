import { useState, useEffect } from 'react'
import { useIntl } from 'react-intl'

import { Layout } from '@/components/layout'
import { messages as menuT } from '@/components/menu/menu-i18n'

export const Things = () => {
  const [loading, setLoading] = useState(true)
  const { formatMessage: _ } = useIntl()

  // Demo for showing the loader for 2 seconds after page visit
  useEffect(() => {
    const timeout = setTimeout(() => {
      setLoading(false)
    }, 2000)

    return () => {
      clearTimeout(timeout)
    }
  }, [])

  return (
    <Layout
      title={_(menuT.things)}
      breadcrumbs={[
        {
          to: '/',
          label: _(menuT.dashboard),
        },
        {
          label: _(menuT.things),
        },
      ]}
      loading={loading}
      header={<div><i>{'Filter goes here'}</i></div>}
    >
      {<i>{'Table goes here'}</i>}
    </Layout>
  )
}
