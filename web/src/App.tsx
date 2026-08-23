import { useState } from 'react'
import { FlightMap, FlightSidebar, AltitudeLegend, useFlights, useFlightRecord} from './features/flights'

function App() {
  const { flights, count, error, lastUpdated } = useFlights(5000)
  const [selected, setSelected] = useState<string | null>(null)
  const { data: record, loading: recordLoading, error: recordError } = useFlightRecord(selected)

  return (
    <div className="relative h-full w-full">
      <FlightMap
        flights={flights}
        track={record?.track}
        onSelect={setSelected}
      />

      <div className="absolute flex flex-col items-center left-3 top-3 z-10 rounded-lg bg-gray-900 px-4 py-2 text-white shadow-xl backdrop-blur">
        <div className="text-sm font-semibold">- live traffic -</div>
        <div className="text-xs text-gray-300">
          {error ? (
            <span className="text-red-400">error: {error}</span>
          ) : (
            <>
              {count.toLocaleString()} active -{' '}
              {lastUpdated
                ? new Date(lastUpdated).toLocaleTimeString()
                : 'loading…'}
            </>
          )}
        </div>
      </div>

      <AltitudeLegend />

      {selected && (
        <FlightSidebar
          record={record}
          loading={recordLoading}
          error={recordError}
          onClose={() => setSelected(null)}
        />
      )}
    </div>
  )
}

export default App
