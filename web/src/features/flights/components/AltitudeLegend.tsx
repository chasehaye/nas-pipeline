// Mirrors the icon-color ramp in FlightMap: gray (ground) -> orange -> purple.
const GRADIENT =
  'linear-gradient(to top, ' +
  '#9ca3af 0%, #f97316 2%, #fb7185 20%, #ec4899 40%, ' +
  '#d946ef 60%, #a855f7 80%, #7c3aed 100%)'

const BAR_HEIGHT = '7rem'

export function AltitudeLegend() {
  return (
    <div className="absolute bottom-3 left-3 z-10 rounded-lg bg-gray-900/80 px-3 py-2 text-white shadow-lg backdrop-blur">
      <div className="mb-1 text-xs font-semibold">Altitude</div>
      <div className="flex gap-2">
        <div
          className="w-3 rounded"
          style={{ height: BAR_HEIGHT, background: GRADIENT }}
        />
        <div
          className="flex flex-col justify-between text-[10px] text-gray-300"
          style={{ height: BAR_HEIGHT }}
        >
          <span>50k</span>
          <span>40k</span>
          <span>30k</span>
          <span>20k</span>
          <span>10k</span>
          <span>gnd</span>
        </div>
      </div>
    </div>
  )
}
