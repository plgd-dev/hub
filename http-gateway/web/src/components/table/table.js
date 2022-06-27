import { useEffect } from 'react'
import { useTable, useSortBy, usePagination, useRowSelect } from 'react-table'
import BTable from 'react-bootstrap/Table'
import PropTypes from 'prop-types'
import classNames from 'classnames'

import { Pagination } from './pagination'
import { compareIgnoreCase } from './utils'

const defaultPropGetter = () => ({})

export const Table = ({
  className,
  columns,
  data,
  onRowsSelect,
  primaryAttribute,
  defaultSortBy,
  defaultPageSize,
  autoFillEmptyRows,
  getRowProps = defaultPropGetter,
  getColumnProps = defaultPropGetter,
  getCellProps = defaultPropGetter,
  paginationProps,
  enablePagination,
  bottomControls,
  unselectRowsToken,
}) => {
  const {
    getTableProps,
    getTableBodyProps,
    headerGroups,
    prepareRow,
    page,

    canPreviousPage,
    canNextPage,
    pageOptions,
    pageCount,
    gotoPage,
    nextPage,
    previousPage,
    setPageSize,

    selectedFlatRows,
    toggleAllRowsSelected,

    state: { pageIndex, pageSize, selectedRowIds },
  } = useTable(
    {
      columns,
      data,
      initialState: {
        sortBy: defaultSortBy,
        pageSize: defaultPageSize,
      },
      sortTypes: {
        alphanumeric: (row1, row2, columnName) => {
          if (
            row1 &&
            row1.values[columnName] &&
            row2 &&
            row2.values[columnName]
          ) {
            return compareIgnoreCase(
              row1.values[columnName],
              row2.values[columnName]
            )
          }

          // fix metedata.status.value: undefined
          return false
        },
      },
      autoResetPage: false,
      autoResetSelectedRows: false,
    },
    useSortBy,
    usePagination,
    useRowSelect
  )

  // Calls the onRowsSelect handler after a row was selected/unselected,
  // so that the parent can store the current selection.
  useEffect(() => {
    if (onRowsSelect && selectedRowIds && primaryAttribute) {
      onRowsSelect(selectedFlatRows.map(d => d.original[primaryAttribute]))
    }
  }, [selectedRowIds, primaryAttribute]) // eslint-disable-line

  // Any time the unselectRowsToken is changed, all rows are gonna be unselected
  useEffect(() => {
    toggleAllRowsSelected(false)
  }, [unselectRowsToken]) // eslint-disable-line

  // When the defaultPageSize is changed, update the pageSize in the table
  useEffect(() => {
    setPageSize(defaultPageSize)
  }, [defaultPageSize]) // eslint-disable-line

  return (
    <>
      <div className={classNames('plgd-table', className)}>
        <BTable responsive striped {...getTableProps()}>
          <thead>
            {headerGroups.map(headerGroup => (
              <tr {...headerGroup.getHeaderGroupProps()}>
                {headerGroup.headers.map(column => (
                  // Sorting props to control sorting
                  <th
                    {...column.getHeaderProps(column.getSortByToggleProps())}
                    style={{
                      ...column.getHeaderProps(column.getSortByToggleProps())
                        .style,
                      ...column.style,
                    }}
                    className={classNames(
                      column.getHeaderProps(column.getSortByToggleProps())
                        .className,
                      column.className
                    )}
                  >
                    <div className="th-div">
                      {column.render('Header')}
                      {column.canSort && (
                        <span
                          className={classNames('sort-arrows', {
                            desc: column.isSorted && column.isSortedDesc,
                            asc: column.isSorted && !column.isSortedDesc,
                          })}
                        >
                          <i className="fas fa-caret-up icon-asc" />
                          <i className="fas fa-caret-down icon-desc" />
                        </span>
                      )}
                    </div>
                  </th>
                ))}
              </tr>
            ))}
          </thead>
          <tbody {...getTableBodyProps()}>
            {page.map(row => {
              prepareRow(row)
              return (
                <tr {...row.getRowProps(getRowProps(row))}>
                  {row.cells.map(cell => {
                    return (
                      <td
                        {...cell.getCellProps([
                          {
                            className: cell.column.className,
                            style: cell.column.style,
                          },
                          getColumnProps(cell.column),
                          getCellProps(cell),
                        ])}
                      >
                        {cell.render('Cell')}
                      </td>
                    )
                  })}
                </tr>
              )
            })}
            {autoFillEmptyRows &&
              page.length < pageSize &&
              Array(pageSize - page.length)
                .fill(0)
                .map((emptyRow, i) => {
                  return (
                    <tr key={`empty-table-row-${i}`}>
                      <td colSpan={100} />
                    </tr>
                  )
                })}
          </tbody>
        </BTable>
      </div>

      <div className="table-bottom-controls">
        {bottomControls || <div />}
        {pageCount > 0 && enablePagination && (
          <Pagination
            {...paginationProps}
            canPreviousPage={canPreviousPage}
            canNextPage={canNextPage}
            pageOptions={pageOptions}
            pageCount={pageCount}
            gotoPage={gotoPage}
            nextPage={nextPage}
            previousPage={previousPage}
            setPageSize={setPageSize}
            pageIndex={pageIndex}
            pageSize={pageSize}
            pageLength={page.length}
          />
        )}
      </div>
    </>
  )
}

Table.propTypes = {
  data: PropTypes.arrayOf(PropTypes.object).isRequired,
  columns: PropTypes.arrayOf(
    PropTypes.shape({
      accessor: PropTypes.oneOfType([PropTypes.string, PropTypes.func])
        .isRequired,
      id: PropTypes.string, // required if accessor is function
      columns: PropTypes.array,
      Header: PropTypes.oneOfType([
        PropTypes.string,
        PropTypes.func,
        PropTypes.element,
      ]),
      Footer: PropTypes.oneOfType([
        PropTypes.string,
        PropTypes.func,
        PropTypes.element,
      ]),
      Cell: PropTypes.oneOfType([PropTypes.func, PropTypes.element]),
      width: PropTypes.number,
      minWidth: PropTypes.number,
      maxWidth: PropTypes.number,
    })
  ).isRequired,
  defaultSortBy: PropTypes.arrayOf(
    PropTypes.shape({
      id: PropTypes.string,
      desc: PropTypes.bool,
    })
  ),
  defaultPageSize: PropTypes.number,
  autoFillEmptyRows: PropTypes.bool, // Fill empty rows to match the pageSize (to keep the table always the same size)
  className: PropTypes.string,
  paginationProps: PropTypes.object,
  onRowsSelect: PropTypes.func,
  primaryAttribute: PropTypes.string,
  bottomControls: PropTypes.node,
  unselectRowsToken: PropTypes.oneOfType([PropTypes.string, PropTypes.number]),
  enablePagination: PropTypes.bool,
}

Table.defaultProps = {
  defaultSortBy: [],
  defaultPageSize: 10,
  autoFillEmptyRows: false,
  className: null,
  paginationProps: {},
  onRowsSelect: null,
  primaryAttribute: null,
  bottomControls: null,
  unselectRowsToken: null,
  enablePagination: true,
}
