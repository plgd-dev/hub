import { useTable, useExpanded } from 'react-table'
import BTable from 'react-bootstrap/Table'
import PropTypes from 'prop-types'
import classNames from 'classnames'

const defaultPropGetter = () => ({})

export const TreeTable = ({
  className,
  columns,
  data,
  getRowProps = defaultPropGetter,
}) => {
  const {
    getTableProps,
    getTableBodyProps,
    headerGroups,
    prepareRow,
    rows,
  } = useTable(
    {
      columns,
      data,
      autoResetExpanded: false,
    },
    useExpanded
  )

  return (
    <>
      <div className={classNames('plgd-table', 'tree-table', className)}>
        <BTable responsive striped {...getTableProps()}>
          <thead>
            {headerGroups.map(headerGroup => (
              <tr {...headerGroup.getHeaderGroupProps()}>
                {headerGroup.headers.map(column => (
                  <th
                    {...column.getHeaderProps()}
                    style={{
                      ...column.getHeaderProps().style,
                      ...column.style,
                    }}
                  >
                    {column.render('Header')}
                  </th>
                ))}
              </tr>
            ))}
          </thead>
          <tbody {...getTableBodyProps()}>
            {rows.map(row => {
              prepareRow(row)
              return (
                <tr {...row.getRowProps(getRowProps(row))}>
                  {row.cells.map(cell => {
                    return (
                      <td {...cell.getCellProps()}>{cell.render('Cell')}</td>
                    )
                  })}
                </tr>
              )
            })}
          </tbody>
        </BTable>
      </div>
    </>
  )
}

TreeTable.propTypes = {
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
  className: PropTypes.string,
}

TreeTable.defaultProps = {
  className: null,
}
