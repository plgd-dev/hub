import { useTable, useSortBy, usePagination } from 'react-table'
import BTable from 'react-bootstrap/Table'
import PropTypes from 'prop-types'
import classNames from 'classnames'

import { Pagination } from './pagination'

export const Table = ({
  columns,
  data,
  defaultSortBy,
  defaultPageSize,
  autoFillEmptyRows,
}) => {
  const {
    getTableProps,
    getTableBodyProps,
    headerGroups,
    prepareRow,
    page,

    // Pagination
    canPreviousPage,
    canNextPage,
    pageOptions,
    pageCount,
    gotoPage,
    nextPage,
    previousPage,
    setPageSize,
    state: { pageIndex, pageSize },
  } = useTable(
    {
      columns,
      data,
      initialState: {
        sortBy: defaultSortBy,
        pageSize: defaultPageSize,
      },
    },
    useSortBy,
    usePagination
  )

  return (
    <>
      <div className="plgd-table">
        <BTable responsive striped {...getTableProps()}>
          <thead>
            {headerGroups.map(headerGroup => (
              <tr {...headerGroup.getHeaderGroupProps()}>
                {headerGroup.headers.map(column => (
                  // Sorting props to control sorting
                  <th {...column.getHeaderProps(column.getSortByToggleProps())}>
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
                <tr {...row.getRowProps()}>
                  {row.cells.map(cell => {
                    return (
                      <td {...cell.getCellProps()}>{cell.render('Cell')}</td>
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

      {page.length > 0 && (
        <div className="table-bottom-controls">
          <div />
          <Pagination
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
          />
        </div>
      )}
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
}

Table.defaultProps = {
  defaultSortBy: [],
  defaultPageSize: 10,
  autoFillEmptyRows: false,
}
