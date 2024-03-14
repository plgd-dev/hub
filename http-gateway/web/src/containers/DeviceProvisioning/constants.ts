export const dpsApiEndpoints = {
    PROVISIONING_RECORDS: '/api/v1/provisioning-records',
    ENROLLMENT_GROUPS: '/api/v1/enrollment-groups',
    HUBS: '/api/v1/hubs',
}

export const DPS_DEFAULT_PAGE_SIZE = 10

// Maximum amount of provisioningRecordsIds filled into one delete request.
// (if ther is more provisioningRecordsIds then the provided number, it creates more chunks of a maximum of this number)
export const DPS_DELETE_CHUNK_SIZE = 50

export const provisioningStatuses = {
    ERROR: 'error',
    SUCCESS: 'success',
    WARNING: 'warning',
} as const
