const testId = {
    app: {
        logout: 'hub-app-logout',
    },
    menu: {
        devices: 'hub-menu-item-devices',
    },
    devices: {
        detail: {
            layout: 'hub-devices-detail-layout',
            tabInformation: 'hub-devices-detail-tab-information',
            tabResources: 'hub-devices-detail-tab-resources',
            tabCertificates: 'hub-devices-detail-tab-certificates',
            tabProvisioningRecords: 'hub-devices-detail-tab-provisioning-records',
            informationTableId: 'hub-devices-detail-information-table-id',
            editNameButton: 'hub-devices-detail-edit-name-button',
            editNameModal: 'hub-devices-detail-edit-name-modal',
            deleteDeviceButton: 'hub-devices-detail-delete-device-button',
            deleteDeviceModal: 'hub-devices-detail-delete-device-modal',
            deleteDeviceButtonCancel: 'hub-devices-detail-delete-device-button-cancel',
            deleteDeviceButtonDelete: 'hub-devices-detail-delete-device-button-delete',
            information: {
                twinToggle: 'hub-devices-detail-information-twin-toggle',
                notificationsToggle: 'hub-devices-detail-information-notifications-toggle',
                endpoints: 'hub-devices-detail-information-endpoints',
                types: 'hub-devices-detail-information-types',
            },
            resources: {
                table: 'hub-devices-detail-resources-table',
                tree: 'hub-devices-detail-resources-tree',
                updateModal: 'hub-devices-detail-resources-update-modal',
                viewSwitch: 'hub-devices-detail-resources-view-switch',
                deleteModal: 'hub-devices-detail-resources-delete-modal',
            },
        },
    },
    remoteClients: {
        detail: {
            tabInformation: 'hub-remote-clients-detail-tab-information',
            tabConfiguration: 'hub-remote-clients-detail-tab-configuration',
        },
    },
    dps: {
        provisioningRecords: {
            detail: {
                deleteButton: 'hub-dps-provisioning-records-detail-delete-button',
                editNameButton: 'hub-dps-provisioning-records-detail-edit-name-button',
                deleteButtonCancel: 'hub-dps-provisioning-records-detail-delete-button-cancel',
                deleteButtonConfirm: 'hub-dps-provisioning-records-detail-delete-button-confirm',
                editNameModal: 'hub-dps-provisioning-records-detail-edit-name-modal',
                tabDetails: 'hub-dps-provisioning-records-detail-tab-details',
                tabCredentials: 'hub-dps-provisioning-records-detail-tab-credentials',
                tabAcls: 'hub-dps-provisioning-records-detail-tab-acls',
            },
        },
        enrollmentGroups: {
            detail: {
                tabEnrollmentConfiguration: 'hub-dps-enrollment-groups-detail-tab-enrollment-configuration',
                tabDeviceCredentials: 'hub-dps-enrollment-groups-detail-tab-device-credentials',
                deleteButton: 'hub-dps-enrollment-groups-detail-delete-button',
                deleteButtonCancel: 'hub-dps-enrollment-groups-detail-delete-button-cancel',
                deleteButtonConfirm: 'hub-dps-enrollment-groups-detail-delete-button-confirm',
                editNameButton: 'hub-dps-enrollment-groups-detail-edit-name-button',
                editNameModal: 'hub-dps-enrollment-groups-detail-edit-name-modal',
            },
        },
        linkedHubs: {
            detail: {
                tabDetails: 'hub-dps-linked-hubs-detail-tab-details',
                tabCertificateAuthority: 'hub-dps-linked-hubs-detail-tab-certificate-authority',
                tabAuthorization: 'hub-dps-linked-hubs-detail-tab-Authorization',
                deleteButton: 'hub-dps-linked-hubs-detail-delete-button',
                editNameButton: 'hub-dps-linked-hubs-detail-edit-name-button',
                deleteButtonCancel: 'hub-dps-linked-hubs-detail-delete-button-cancel',
                deleteButtonConfirm: 'hub-dps-linked-hubs-detail-delete-button-confirm',
                editNameModal: 'hub-dps-linked-hubs-detail-edit-name-modal',
            },
        },
        certificates: {
            detail: {
                tabCertificateAuthorityConfiguration: 'hub-dps-certificates-detail-tab-certificate-authority-configuration',
                tabAuthorization: 'hub-dps-certificates-detail-tab-Authorization',
                deleteButton: 'hub-dps-certificates-detail-delete-button',
                editNameButton: 'hub-dps-certificates-detail-edit-name-button',
                deleteButtonCancel: 'hub-dps-certificates-detail-delete-button-cancel',
                deleteButtonConfirm: 'hub-dps-certificates-detail-delete-button-confirm',
                editNameModal: 'hub-dps-certificates-detail-edit-name-modal',
            },
        },
    },
    snippetService: {
        configurations: {
            detail: {
                deleteButton: 'hub-snippet-service-configurations-detail-delete-button',
                deleteButtonConfirm: 'hub-snippet-service-configurations-detail-delete-button-confirm',
                deleteButtonCancel: 'hub-snippet-service-configurations-detail-delete-button-cancel',
                tabGeneral: 'hub-snippet-service-configurations-detail-tab-general',
                tabConditions: 'hub-snippet-service-configurations-detail-tab-conditions',
                tabDeviceAppliedConfiguration: 'hub-snippet-service-configurations-detail-tab-device-applied-configuration',
            },
        },
    },
}

export default testId
