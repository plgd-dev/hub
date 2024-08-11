import React, { FC, useCallback, useState } from 'react'
import isFunction from 'lodash/isFunction'

import Button from '@shared-ui/components/Atomic/Button'
import { IconTrash } from '@shared-ui/components/Atomic/Icon'
import DeleteModal from '@shared-ui/components/Atomic/Modal/components/DeleteModal'

import { Props } from './DetailHeaderLayout.types'

const DetailHeaderLayout: FC<Props> = (props) => {
    const { customButton, deleteApiMethod, deleteInformation, id, i18n, loading, onDeleteSuccess, onDeleteError, testIds } = props

    const [deleteModal, setDeleteModal] = useState(false)
    const [deleting, setDeleting] = useState(false)

    const handleDelete = useCallback(async () => {
        try {
            setDeleting(true)

            if (isFunction(deleteApiMethod)) {
                await deleteApiMethod([id])
            }

            setDeleting(false)
            setDeleteModal(false)

            isFunction(onDeleteSuccess) && onDeleteSuccess()
        } catch (e: any) {
            setDeleting(false)
            setDeleteModal(false)

            isFunction(onDeleteError) && onDeleteError(e)
        }
    }, [deleteApiMethod, id, onDeleteError, onDeleteSuccess])

    return (
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
            {customButton}
            {isFunction(deleteApiMethod) && (
                <Button
                    dataTestId={testIds?.deleteButton}
                    disabled={loading}
                    htmlType='button'
                    icon={<IconTrash />}
                    onClick={() => setDeleteModal(true)}
                    variant='tertiary'
                >
                    {i18n.delete}
                </Button>
            )}
            {isFunction(deleteApiMethod) && (
                <DeleteModal
                    dataTestId={testIds?.deleteModal}
                    deleteInformation={deleteInformation}
                    footerActions={[
                        {
                            dataTestId: testIds?.deleteButtonCancel,
                            label: i18n.cancel,
                            onClick: () => setDeleteModal(false),
                            variant: 'tertiary',
                        },
                        {
                            dataTestId: testIds?.deleteButtonConfirm,
                            label: i18n.delete,
                            loading: deleting,
                            loadingText: i18n.deleting,
                            onClick: handleDelete,
                            variant: 'primary',
                        },
                    ]}
                    onClose={() => setDeleteModal(false)}
                    show={deleteModal}
                    subTitle={i18n.subTitle}
                    title={i18n.title}
                />
            )}
        </div>
    )
}

DetailHeaderLayout.displayName = 'DetailHeaderLayout'

export default DetailHeaderLayout
