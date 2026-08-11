import { FlightMap, useFlights } from './features/flights'

function App() {
  const { flights, count, error, lastUpdated } = useFlights(4000)

  return (
    <div className="relative h-full w-full">
      <FlightMap flights={flights} />

      <div className="absolute flex flex-col items-center left-3 top-3 z-10 rounded-lg bg-gray-900/80 px-4 py-2 text-white shadow-lg backdrop-blur">
        <div className="text-sm font-semibold">- live traffic -</div>
        <div className="text-xs text-gray-300">
          {error ? (
            <span className="text-red-400">error: {error}</span>
          ) : (
            <>
              {count.toLocaleString()} active ·{' '}
              {lastUpdated
                ? new Date(lastUpdated).toLocaleTimeString()
                : 'loading…'}
            </>
          )}
        </div>
      </div>
    </div>
  )
}

export default App
