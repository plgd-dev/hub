import { FC } from 'react'
import { useIntl } from 'react-intl'
import { generatePath, useNavigate } from 'react-router-dom'

import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import { getApiErrorMessage } from '@shared-ui/common/utils'

import { messages as g } from '@/containers/Global.i18n'
import DetailHeaderLayout from '@/containers/Common/DetailHeaderLayout'
import { messages as confT } from '../../SnippetService.i18n'
import { deleteResourcesConfigApi } from '@/containers/SnippetService/rest'
import testId from '@/testId'
import notificationId from '@/notificationId'
import { pages } from '@/routes'
import { DeleteInformationType } from '@shared-ui/components/Atomic/Modal/components/DeleteModal/DeleteModal.types'

type Props = {
    id: string
    loading: boolean
    name?: string
}

const DetailHeader: FC<Props> = (props) => {
    const { formatMessage: _ } = useIntl()

    const navigate = useNavigate()

    return (
        <DetailHeaderLayout
            deleteApiMethod={deleteResourcesConfigApi}
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
                subTitle: _(confT.deleteConditionMessage),
                title: _(confT.deleteCondition),
            }}
            id={props.id}
            loading={props.loading}
            onDeleteError={(error: any) => {
                Notification.error(
                    { title: _(confT.conditionsError), message: getApiErrorMessage(error) },
                    { notificationId: notificationId.HUB_SNIPPET_SERVICE_CONDITIONS_DETAIL_PAGE_DELETE_ERROR }
                )
            }}
            onDeleteSuccess={() => {
                Notification.success(
                    { title: _(confT.conditionDeleted), message: _(confT.conditionsDeletedMessage) },
                    { notificationId: notificationId.HUB_SNIPPET_SERVICE_CONDITIONS_DETAIL_PAGE_DELETE_SUCCESS }
                )

                navigate(generatePath(pages.CONDITIONS.CONDITIONS.LINK))
            }}
            testIds={{
                deleteButton: testId.snippetService.resourcesConfig.detail.deleteButton,
                deleteButtonCancel: testId.snippetService.resourcesConfig.detail.deleteButtonCancel,
                deleteButtonConfirm: testId.snippetService.resourcesConfig.detail.deleteButtonConfirm,
            }}
        />
    )
}

DetailHeader.displayName = 'DetailHeader'

export default DetailHeader
