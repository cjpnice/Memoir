import type { Project, ProjectPlace } from "@/lib/types";

type CityEntry = ProjectPlace & {
  aliases?: string[];
};

const cityCatalog: CityEntry[] = [
  { city: "北京", region: "北京", country: "中国", latitude: 39.9042, longitude: 116.4074, source: "city_catalog", confidence: 0.96, aliases: ["北京市", "beijing", "peking"] },
  { city: "上海", region: "上海", country: "中国", latitude: 31.2304, longitude: 121.4737, source: "city_catalog", confidence: 0.96, aliases: ["上海市", "shanghai"] },
  { city: "广州", region: "广东", country: "中国", latitude: 23.1291, longitude: 113.2644, source: "city_catalog", confidence: 0.95, aliases: ["广州市", "guangzhou", "canton"] },
  { city: "深圳", region: "广东", country: "中国", latitude: 22.5431, longitude: 114.0579, source: "city_catalog", confidence: 0.95, aliases: ["深圳市", "shenzhen"] },
  { city: "杭州", region: "浙江", country: "中国", latitude: 30.2741, longitude: 120.1551, source: "city_catalog", confidence: 0.95, aliases: ["杭州市", "hangzhou"] },
  { city: "南京", region: "江苏", country: "中国", latitude: 32.0603, longitude: 118.7969, source: "city_catalog", confidence: 0.94, aliases: ["南京市", "nanjing"] },
  { city: "苏州", region: "江苏", country: "中国", latitude: 31.2989, longitude: 120.5853, source: "city_catalog", confidence: 0.94, aliases: ["苏州市", "suzhou"] },
  { city: "成都", region: "四川", country: "中国", latitude: 30.5728, longitude: 104.0668, source: "city_catalog", confidence: 0.94, aliases: ["成都市", "chengdu"] },
  { city: "重庆", region: "重庆", country: "中国", latitude: 29.563, longitude: 106.5516, source: "city_catalog", confidence: 0.94, aliases: ["重庆市", "chongqing"] },
  { city: "西安", region: "陕西", country: "中国", latitude: 34.3416, longitude: 108.9398, source: "city_catalog", confidence: 0.94, aliases: ["西安市", "xian", "xi'an"] },
  { city: "武汉", region: "湖北", country: "中国", latitude: 30.5928, longitude: 114.3055, source: "city_catalog", confidence: 0.94, aliases: ["武汉市", "wuhan"] },
  { city: "厦门", region: "福建", country: "中国", latitude: 24.4798, longitude: 118.0894, source: "city_catalog", confidence: 0.94, aliases: ["厦门市", "xiamen"] },
  { city: "青岛", region: "山东", country: "中国", latitude: 36.0671, longitude: 120.3826, source: "city_catalog", confidence: 0.94, aliases: ["青岛市", "qingdao"] },
  { city: "天津", region: "天津", country: "中国", latitude: 39.3434, longitude: 117.3616, source: "city_catalog", confidence: 0.93, aliases: ["天津市", "tianjin"] },
  { city: "长沙", region: "湖南", country: "中国", latitude: 28.2282, longitude: 112.9388, source: "city_catalog", confidence: 0.93, aliases: ["长沙市", "changsha"] },
  { city: "昆明", region: "云南", country: "中国", latitude: 25.0389, longitude: 102.7183, source: "city_catalog", confidence: 0.93, aliases: ["昆明市", "kunming"] },
  { city: "大理", region: "云南", country: "中国", latitude: 25.6065, longitude: 100.2676, source: "city_catalog", confidence: 0.93, aliases: ["大理市", "dali"] },
  { city: "丽江", region: "云南", country: "中国", latitude: 26.8721, longitude: 100.2296, source: "city_catalog", confidence: 0.93, aliases: ["丽江市", "lijiang"] },
  { city: "桂林", region: "广西", country: "中国", latitude: 25.2736, longitude: 110.2902, source: "city_catalog", confidence: 0.93, aliases: ["桂林市", "guilin"] },
  { city: "三亚", region: "海南", country: "中国", latitude: 18.2528, longitude: 109.5119, source: "city_catalog", confidence: 0.93, aliases: ["三亚市", "sanya"] },
  { city: "海口", region: "海南", country: "中国", latitude: 20.044, longitude: 110.1999, source: "city_catalog", confidence: 0.92, aliases: ["海口市", "haikou"] },
  { city: "拉萨", region: "西藏", country: "中国", latitude: 29.652, longitude: 91.1721, source: "city_catalog", confidence: 0.92, aliases: ["拉萨市", "lhasa"] },
  { city: "乌鲁木齐", region: "新疆", country: "中国", latitude: 43.8256, longitude: 87.6168, source: "city_catalog", confidence: 0.92, aliases: ["乌鲁木齐市", "urumqi"] },
  { city: "哈尔滨", region: "黑龙江", country: "中国", latitude: 45.8038, longitude: 126.5349, source: "city_catalog", confidence: 0.92, aliases: ["哈尔滨市", "harbin"] },
  { city: "沈阳", region: "辽宁", country: "中国", latitude: 41.8057, longitude: 123.4315, source: "city_catalog", confidence: 0.92, aliases: ["沈阳市", "shenyang"] },
  { city: "香港", region: "香港", country: "中国", latitude: 22.3193, longitude: 114.1694, source: "city_catalog", confidence: 0.95, aliases: ["香港特别行政区", "hong kong", "hongkong"] },
  { city: "澳门", region: "澳门", country: "中国", latitude: 22.1987, longitude: 113.5439, source: "city_catalog", confidence: 0.95, aliases: ["澳门特别行政区", "macau", "macao"] },
  { city: "台北", region: "台湾", country: "中国", latitude: 25.033, longitude: 121.5654, source: "city_catalog", confidence: 0.94, aliases: ["臺北", "taipei"] },
  { city: "东京", region: "关东", country: "日本", latitude: 35.6762, longitude: 139.6503, source: "city_catalog", confidence: 0.94, aliases: ["tokyo"] },
  { city: "京都", region: "京都府", country: "日本", latitude: 35.0116, longitude: 135.7681, source: "city_catalog", confidence: 0.94, aliases: ["kyoto"] },
  { city: "大阪", region: "大阪府", country: "日本", latitude: 34.6937, longitude: 135.5023, source: "city_catalog", confidence: 0.94, aliases: ["osaka"] },
  { city: "首尔", region: "首尔", country: "韩国", latitude: 37.5665, longitude: 126.978, source: "city_catalog", confidence: 0.93, aliases: ["seoul"] },
  { city: "曼谷", region: "曼谷", country: "泰国", latitude: 13.7563, longitude: 100.5018, source: "city_catalog", confidence: 0.93, aliases: ["bangkok"] },
  { city: "新加坡", region: "新加坡", country: "新加坡", latitude: 1.3521, longitude: 103.8198, source: "city_catalog", confidence: 0.94, aliases: ["singapore"] },
  { city: "巴黎", region: "法兰西岛", country: "法国", latitude: 48.8566, longitude: 2.3522, source: "city_catalog", confidence: 0.93, aliases: ["paris"] },
  { city: "伦敦", region: "英格兰", country: "英国", latitude: 51.5072, longitude: -0.1276, source: "city_catalog", confidence: 0.93, aliases: ["london"] },
  { city: "纽约", region: "纽约州", country: "美国", latitude: 40.7128, longitude: -74.006, source: "city_catalog", confidence: 0.93, aliases: ["new york", "nyc"] },
  { city: "洛杉矶", region: "加利福尼亚", country: "美国", latitude: 34.0522, longitude: -118.2437, source: "city_catalog", confidence: 0.93, aliases: ["los angeles", "la"] },
  { city: "旧金山", region: "加利福尼亚", country: "美国", latitude: 37.7749, longitude: -122.4194, source: "city_catalog", confidence: 0.93, aliases: ["san francisco", "sf"] },
  { city: "悉尼", region: "新南威尔士", country: "澳大利亚", latitude: -33.8688, longitude: 151.2093, source: "city_catalog", confidence: 0.93, aliases: ["sydney"] },
];

const suffixPattern =
  /(特别行政区|自治州|自治区|地区|市区|城市|省|市|县|区|府|州|郡|都|道)$/g;

function stripLocationText(value: string) {
  return value
    .trim()
    .toLowerCase()
    .replace(/[·•,，。.\s_-]+/g, "")
    .replace(/臺/g, "台")
    .replace(suffixPattern, "");
}

export function normalizeLocationText(value: string) {
  let normalized = stripLocationText(value);
  for (let i = 0; i < 3; i += 1) {
    const next = normalized.replace(suffixPattern, "");
    if (next === normalized) break;
    normalized = next;
  }
  return normalized;
}

function clonePlace(entry: ProjectPlace): ProjectPlace {
  return {
    city: entry.city,
    region: entry.region,
    country: entry.country,
    latitude: entry.latitude,
    longitude: entry.longitude,
    source: entry.source,
    confidence: entry.confidence,
  };
}

const cityIndex = new Map<string, ProjectPlace>();

cityCatalog.forEach((entry) => {
  [entry.city, ...(entry.aliases ?? [])]
    .filter(Boolean)
    .forEach((name) => {
      cityIndex.set(normalizeLocationText(name), clonePlace(entry));
    });
});

export function resolveCityPlace(location: string): ProjectPlace | null {
  const normalized = normalizeLocationText(location);
  if (!normalized) return null;
  const exact = cityIndex.get(normalized);
  if (exact) return clonePlace(exact);

  for (const [key, place] of cityIndex.entries()) {
    if (key && normalized.includes(key)) {
      return clonePlace(place);
    }
  }
  return null;
}

export function resolveProjectPlace(project: Project): ProjectPlace | null {
  if (project.place) return project.place;
  return resolveCityPlace(project.location ?? "");
}

export function formatPlaceLabel(place: ProjectPlace) {
  return [place.city, place.region && place.region !== place.city ? place.region : "", place.country]
    .filter(Boolean)
    .join(" · ");
}
