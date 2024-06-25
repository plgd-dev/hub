import { FC } from 'react'
import { useIntl } from 'react-intl'
import { generatePath, useNavigate } from 'react-router-dom'

import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import { getApiErrorMessage } from '@shared-ui/common/utils'
import { DeleteInformationType } from '@shared-ui/components/Atomic/Modal/components/DeleteModal/DeleteModal.types'

import { messages as g } from '@/containers/Global.i18n'
import DetailHeaderLayout from '@/containers/Common/DetailHeaderLayout'
import { messages as confT } from '../../SnippetService.i18n'
import { deleteConfigurationsApi } from '@/containers/SnippetService/rest'
import testId from '@/testId'
import notificationId from '@/notificationId'
import { pages } from '@/routes'

type Props = {
    id: string
    refresh: () => void
    loading?: boolean
    name?: string
}

const DetailHeader: FC<Props> = (props) => {
    const { formatMessage: _ } = useIntl()

    const navigate = useNavigate()

    return (
        <DetailHeaderLayout
            deleteApiMethod={deleteConfigurationsApi}
            deleteInformation={
                [props.name ? { label: _(g.name), value: props.name } : undefined, { label: _(g.id), value: props.id }].filter(
                    (i) => !!i
                ) as DeleteInformationType[]
            }
            i18n={{
                id: _(g.id),
                name: _(g.name),
                cancel: _(g.cancel),
                delete: _(g.delete),
                deleting: _(g.deleting),
                subTitle: _(confT.deleteConfigurationMessage),
                title: _(confT.deleteConfigurationTitle),
            }}
            id={props.id}
            loading={false}
            onDeleteError={(error: any) => {
                Notification.error(
                    { title: _(confT.configurationsError), message: getApiErrorMessage(error) },
                    { notificationId: notificationId.HUB_SNIPPET_SERVICE_CONFIGURATIONS_DETAIL_PAGE_DELETE_ERROR }
                )
            }}
            onDeleteSuccess={() => {
                Notification.success(
                    { title: _(confT.configurationsDeleted), message: _(confT.configurationsDeletedMessage) },
                    { notificationId: notificationId.HUB_SNIPPET_SERVICE_CONFIGURATIONS_DETAIL_PAGE_DELETE_SUCCESS }
                )

                navigate(generatePath(pages.SNIPPET_SERVICE.CONFIGURATIONS.LINK))
            }}
            testIds={{
                deleteButton: testId.snippetService.configurations.detail.deleteButton,
                deleteButtonCancel: testId.snippetService.configurations.detail.deleteButtonCancel,
                deleteButtonConfirm: testId.snippetService.configurations.detail.deleteButtonConfirm,
            }}
        />
    )
}

DetailHeader.displayName = 'DetailHeader'

export default DetailHeader
