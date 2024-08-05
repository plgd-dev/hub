import { FC } from 'react'
import { useIntl } from 'react-intl'
import { generatePath, useNavigate } from 'react-router-dom'

import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import { getApiErrorMessage } from '@shared-ui/common/utils'
import { DeleteInformationType } from '@shared-ui/components/Atomic/Modal/components/DeleteModal/DeleteModal.types'

import DetailHeaderLayout from '@/containers/Common/DetailHeaderLayout'
import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from '../ApiTokens.i18n'
import notificationId from '@/notificationId'
import { pages } from '@/routes'
import testId from '@/testId'
import { removeApiTokenApi } from '@/containers/ApiTokens/rest'

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
            deleteApiMethod={removeApiTokenApi}
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
                subTitle: _(t.deleteApiTokenMessage),
                title: _(t.deleteApiToken),
            }}
            id={props.id}
            loading={props.loading}
            onDeleteError={(error: any) => {
                Notification.error(
                    { title: _(t.apiTokensError), message: getApiErrorMessage(error) },
                    { notificationId: notificationId.HUB_API_TOKENS_DETAIL_PAGE_DELETE_ERROR }
                )
            }}
            onDeleteSuccess={() => {
                Notification.success(
                    { title: _(t.apiTokenDeleted), message: _(t.apiTokenDeletedMessage) },
                    { notificationId: notificationId.HUB_API_TOKENS_DETAIL_PAGE_DELETE_SUCCESS }
                )

                navigate(generatePath(pages.API_TOKENS.LINK))
            }}
            testIds={{
                deleteButton: testId.apiTokens.detail.deleteButton,
                deleteButtonCancel: testId.apiTokens.detail.deleteButtonCancel,
                deleteButtonConfirm: testId.apiTokens.detail.deleteButtonConfirm,
                deleteModal: testId.apiTokens.detail.deleteModal,
            }}
        />
    )
}

DetailHeader.displayName = 'DetailHeader'

export default DetailHeader
