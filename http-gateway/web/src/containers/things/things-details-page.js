import { useEffect } from 'react'
import { useIntl } from 'react-intl'
import { toast } from 'react-toastify'
import { useParams } from 'react-router-dom'
import Row from 'react-bootstrap/Row'
import Col from 'react-bootstrap/Col'

import { Layout } from '@/components/layout'
import { Badge } from '@/components/badge'
import { Label } from '@/components/label'
import { useApi } from '@/common/hooks'
import { getValue } from '@/common/utils'
import { messages as menuT } from '@/components/menu/menu-i18n'

import { thingsStatuses } from './constants'
import { messages as t } from './things-i18n'

export const ThingsDetailsPage = props => {
  const { formatMessage: _ } = useIntl()
  const { id } = useParams()

  const { data, loading, error } = useApi(
    `https://api.try.plgd.cloud/api/v1/devices/${id}`,
    { audience: 'https://try.plgd.cloud' }
  )

  useEffect(
    () => {
      if (error) {
        toast.error(error?.response?.data?.err || error?.message)
      }
    },
    [error]
  )

  const deviceName = data?.device?.n
  const pageTitle = deviceName || id
  const isOnline = thingsStatuses.ONLINE === data?.status
  
  return (
    <Layout
      title={`${deviceName ? deviceName + ' | ' : ''}${_(menuT.things)}`}
      breadcrumbs={[
        {
          to: '/',
          label: _(menuT.dashboard),
        },
        {
          to: '/things',
          label: _(menuT.things),
        },
        {
          label: pageTitle,
        },
      ]}
      loading={loading}
      header={<div />}
    >
      <h2>{pageTitle}</h2>
      <Row>
        <Col>
          <Label title="ID" inline>
            {getValue(data?.device?.di)}
          </Label>
          <Label title={_(t.types)} inline>
            <div className="align-items-end badges-box-vertical">
              {data?.device?.rt?.map?.(type => (
                <Badge key={type}>
                  {type}
                </Badge>
              ))}
            </div>
          </Label>
        </Col>
        <Col>
          <Label title={_(t.status)} inline>
            <Badge className={isOnline ? 'green' : 'red'}>
              {isOnline ? _(t.online) : _(t.offline)}
            </Badge>
          </Label>
        </Col>
      </Row>
    </Layout>
  )
}
