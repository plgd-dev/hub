import { defineMessages } from '@formatjs/intl'

export const messages = defineMessages({
    enrollmentGroup: {
        id: 'enrollmentGroups.enrollmentGroup',
        defaultMessage: 'Enrollment Group',
    },
    deleteEnrollmentGroupMessage: {
        id: 'enrollmentGroups.deleteEnrollmentGroupMessage',
        defaultMessage: 'Are you sure you want to delete this enrollment group?',
    },
    deleteEnrollmentGroupsMessage: {
        id: 'enrollmentGroups.deleteRecordsMessage',
        defaultMessage: 'Are you sure you want to delete {count} enrollment groups?',
    },
    deleteEnrollmentGroupsSubTitle: {
        id: 'enrollmentGroups.deleteEnrollmentGroupsSubTitle',
        defaultMessage: 'This action cannot be undone.',
    },
    enrollmentGroupsError: {
        id: 'enrollmentGroups.enrollmentGroupsError',
        defaultMessage: 'Enrollment Groups Error',
    },
    addEnrollmentGroup: {
        id: 'enrollmentGroups.addEnrollmentGroup',
        defaultMessage: 'Add Enrollment Group',
    },
    enrollmentConfiguration: {
        id: 'enrollmentGroups.enrollmentConfiguration',
        defaultMessage: 'Enrollment Configuration',
    },
    deviceCredentials: {
        id: 'enrollmentGroups.deviceCredentials',
        defaultMessage: 'Device Credentials',
    },
    deleteEnrollmentGroupTitle: {
        id: 'enrollmentGroups.deleteProvisioningRecordTitle',
        defaultMessage: 'Delete Provisioning Record',
    },
    deviceAuthentication: {
        id: 'enrollmentGroups.deviceAuthentication',
        defaultMessage: 'Device Authentication',
    },
    leadCertificate: {
        id: 'enrollmentGroups.leadCertificate',
        defaultMessage: 'Lead Certificate',
    },
    enableExpiredCertificates: {
        id: 'enrollmentGroups.enableExpiredCertificates',
        defaultMessage: 'Enable Expired Certificates',
    },
    nameError: {
        id: 'enrollmentGroups.nameError',
        defaultMessage: 'Name error',
    },
    fields: {
        id: 'enrollmentGroups.fields',
        defaultMessage: 'fields',
    },
    field: {
        id: 'enrollmentGroups.field',
        defaultMessage: 'field',
    },
    linkedHubs: {
        id: 'enrollmentGroups.linkedHubs',
        defaultMessage: 'Linked Hubs',
    },
    uploadCertDescription: {
        id: 'enrollmentGroups.uploadCertDescription',
        defaultMessage: 'Supported formats: PEM, CRT or CER (max 1 MB)',
    },
    uploadCertTitle: {
        id: 'enrollmentGroups.uploadCertTitle',
        defaultMessage: 'Drag & Drop or Choose file to upload Certificate',
    },
    certificationParsingError: {
        id: 'enrollmentGroups.certificationParsingError',
        defaultMessage: 'Certification Parsing Error',
    },
    certificateDetail: {
        id: 'enrollmentGroups.certificateDetail',
        defaultMessage: 'Certificate Detail',
    },
    preSharedKeySettings: {
        id: 'enrollmentGroups.preSharedKeySettings',
        defaultMessage: 'Pre-Shared key settings',
    },
    preSharedKey: {
        id: 'enrollmentGroups.preSharedKey',
        defaultMessage: 'Pre-Shared key',
    },
    addEnrollmentGroupDescription: {
        id: 'enrollmentGroups.addEnrollmentGroupDescription',
        defaultMessage:
            'An enrollment group is a configuration entry designed for a collection of devices that utilize a common attestation mechanism. It is particularly recommended for managing a large quantity of devices that either share similar initial settings or are allocated to the same tenant. Enrollment groups specifically support X.509 certificates as their attestation method.',
    },
    addEnrollmentGroupDeviceAuthenticationDescription: {
        id: 'enrollmentGroups.addEnrollmentGroupDeviceAuthenticationDescription',
        defaultMessage:
            'An enrollment group is a configuration entry designed for a collection of devices that utilize a common attestation mechanism. The Device Provisioning Service identifies the enrollment group for a device by matching its certificate with the configured attestation mechanism.',
    },
    addEnrollmentGroupDeviceCredentialsDescription: {
        id: 'enrollmentGroups.addEnrollmentGroupDeviceCredentialsDescription',
        defaultMessage:
            'Configured credentials, which typically include the identity certificate and optionally the pre-shared key, enable secure authentication and interaction with the device.',
    },
    enrollmentGroupsDeleted: {
        id: 'enrollmentGroups.provisioningRecordDeleted',
        defaultMessage: 'Enrollment groups deleted',
    },
    enrollmentGroupsDeletedMessage: {
        id: 'enrollmentGroups.provisioningRecordDeletedMessage',
        defaultMessage: 'The selected enrollment groups has been deleted.',
    },
    enrollmentGroupUpdated: {
        id: 'enrollmentGroups.enrollmentGroupUpdated',
        defaultMessage: 'Enrollment group updated',
    },
    enrollmentGroupUpdatedMessage: {
        id: 'enrollmentGroups.enrollmentGroupUpdatedMessage',
        defaultMessage: 'The selected enrollment group has been updated.',
    },
    enrollmentGroupCreated: {
        id: 'enrollmentGroups.enrollmentGroupCreated',
        defaultMessage: 'Enrollment group created',
    },
    enrollmentGroupCreatedMessage: {
        id: 'enrollmentGroups.enrollmentGroupCreatedMessage',
        defaultMessage: 'The enrollment group has been created.',
    },
    tab1Description: {
        id: 'enrollmentGroups.tab1Description',
        defaultMessage: 'Basic setup',
    },
    tab2Description: {
        id: 'enrollmentGroups.tab2Description',
        defaultMessage: 'Certificate Chain Identification',
    },
    tab3Description: {
        id: 'enrollmentGroups.tab3Description',
        defaultMessage: 'Pre-shared Key Configuration',
    },
})
