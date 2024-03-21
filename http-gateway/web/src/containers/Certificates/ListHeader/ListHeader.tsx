import React, { FC, useCallback, useState } from 'react'
import { useIntl } from 'react-intl'

import { IconPlus } from '@shared-ui/components/Atomic'
import Button from '@shared-ui/components/Atomic/Button'
import Modal from '@shared-ui/components/Atomic/Modal'
import Dropzone from '@shared-ui/components/Atomic/Dropzone'
import ButtonBox from '@shared-ui/components/Atomic/ButtonBox/ButtonBox'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import FormLabel from '@shared-ui/components/Atomic/FormLabel'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import { parseCertificate } from '@shared-ui/common/services/certificates'

import { Props } from './ListHeader.types'
import { messages as t } from '../Certificates.i18n'
import { messages as g } from '../../Global.i18n'
import { stringToPem } from '@/containers/DeviceProvisioning/utils'

const ListHeader: FC<Props> = () => {
    const { formatMessage: _ } = useIntl()
    const [addModal, setAddModal] = useState(false)
    const [certFile, setCertFile] = useState<string | undefined>(undefined)
    const [certData, setCertData] = useState<any>(undefined)

    function nameLengthValidator(file: any) {
        const format = file.name.split('.').pop()

        if (!['pem', 'crt', 'cer'].includes(format)) {
            return {
                code: 'invalid-format',
                message: `Bad file format`,
            }
        }
        return null
    }

    const handleCertSave = useCallback(() => {
        setAddModal(false)
    }, [])

    const renderBody = () => (
        <form>
            <FormGroup id='id'>
                <FormLabel text={_(g.name)} />
                <FormInput disabled={true} value={certData?.name || ''} />
            </FormGroup>

            <Dropzone
                smallPadding
                customFileRenders={[{ format: 'pem', icon: 'icon-file-pem' }]}
                description={_(t.uploadCertDescription)}
                maxFiles={1}
                onFilesDrop={(files) => {
                    setTimeout(() => {
                        setCertFile(stringToPem(files[0]))
                        parseCertificate(files[0], 0).then((r) => setCertData(r))
                    }, 100)
                }}
                title={_(t.dndTitle)}
                validator={nameLengthValidator}
            />
            <Spacer type='py-7'>
                <ButtonBox disabled={!certFile || !certData} htmlType='submit' onClick={handleCertSave}>
                    {_(t.saveCertificate)}
                </ButtonBox>
            </Spacer>
        </form>
    )

    return (
        <>
            <Button icon={<IconPlus />} onClick={() => setAddModal(true)} variant='primary'>
                {_(t.certificate)}
            </Button>
            <Modal
                appRoot={document.getElementById('root')}
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
