import React, { FC, useCallback, useState } from 'react'
import { useIntl } from 'react-intl'

import { IconPlus } from '@shared-ui/components/Atomic'
import Button from '@shared-ui/components/Atomic/Button'
import Modal from '@shared-ui/components/Atomic/Modal'

import { Props } from './ListHeader.types'
import { messages as t } from '../Certificates.i18n'
import { messages as g } from '../../../Global.i18n'

const ListHeader: FC<Props> = (props) => {
    const { formatMessage: _ } = useIntl()
    const [addModal, setAddModal] = useState(false)

    const handleAdd = useCallback(() => {
        console.log('handleAdd')
        setAddModal(false)
    }, [])

    const renderBody = () => <div>Body</div>

    return (
        <>
            <Button icon={<IconPlus />} onClick={() => setAddModal(true)} variant='primary'>
                {_(t.certificate)}
            </Button>
            <Modal
                appRoot={document.getElementById('root')}
                footerActions={[
                    {
                        label: _(g.cancel),
                        onClick: () => setAddModal(false),
                        variant: 'tertiary',
                    },
                    {
                        label: _(g.done),
                        onClick: handleAdd,
                        variant: 'primary',
                        // disabled: Object.keys(errors).length > 0,
                    },
                ]}
                maxWidth={600}
                onClose={() => setAddModal(false)}
                portalTarget={document.getElementById('modal-root')}
                renderBody={renderBody}
                show={addModal}
                title={_(t.addCertificate)}
            />
        </>
    )
}

ListHeader.displayName = 'ListHeader'

export default ListHeader
