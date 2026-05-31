"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  BookImage,
  Check,
  LocateFixed,
  MapPin,
  Pencil,
  Route,
  X,
} from "lucide-react";
import MapLibreMap, { Marker, NavigationControl } from 'react-map-gl/maplibre';
import type { MapRef } from 'react-map-gl/maplibre';
import 'maplibre-gl/dist/maplibre-gl.css';
import { DialogShell } from "@/components/DialogShell";
import { formatPlaceLabel, resolveCityPlace, resolveProjectPlace } from "@/lib/cityCoordinates";
import type { ImageAsset, Project, ProjectPlace } from "@/lib/types";

type ProjectPlaceUpdate = {
  location?: string;
  place?: ProjectPlace;
  clearPlace?: boolean;
};

type MemoryMapViewProps = {
  projects: Project[];
  busy: string;
  onSelectProject: (projectId: string) => void;
  onSaveProjectPlace: (projectId: string, input: ProjectPlaceUpdate) => Promise<void>;
  onBack: () => void;
};

type LocatedProject = {
  project: Project;
  place: ProjectPlace;
};

type CityGroup = {
  key: string;
  place: ProjectPlace;
  projects: Project[];
  firstVisitedAt: number;
  imageCount: number;
};

type PlaceDraft = {
  location: string;
  city: string;
  region: string;
  country: string;
  latitude: string;
  longitude: string;
};

type MapStylePreset = {
  id: string;
  name: string;
  url: string;
};

const MAP_STYLES: MapStylePreset[] = [
  {
    id: 'warm',
    name: '暖色调',
    url: 'https://demotiles.maplibre.org/style.json',
  },
  {
    id: 'light',
    name: '浅色',
    url: 'https://tiles.openfreemap.org/styles/liberty',
  },
  {
    id: 'bright',
    name: '明亮',
    url: 'https://tiles.openfreemap.org/styles/bright',
  },
];

const DEFAULT_VIEW = {
  longitude: 104.5,
  latitude: 35.5,
  zoom: 3.2,
};

function formatCount(value: number) {
  return new Intl.NumberFormat("zh-CN").format(value);
}

function timeValue(value?: string) {
  if (!value) return 0;
  const time = new Date(value).getTime();
  return Number.isNaN(time) ? 0 : time;
}

function formatDate(value?: string) {
  if (!value) return "未记录";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "未记录";
  return new Intl.DateTimeFormat("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(date);
}

function cityKey(place: ProjectPlace) {
  return [
    place.country,
    place.region ?? "",
    place.city,
    place.latitude.toFixed(3),
    place.longitude.toFixed(3),
  ].join(":");
}

function projectCoverImage(project: Project): ImageAsset | null {
  const coverId =
    project.album?.pages.find((page) => page.pageType === "cover")?.imageIds[0] ??
    project.album?.pages.find((page) => page.imageIds.length > 0)?.imageIds[0];
  if (coverId) {
    return project.images.find((image) => image.id === coverId) ?? null;
  }
  return project.images[0] ?? null;
}

function imageDisplayUrl(image: ImageAsset) {
  return image.derivedUrl || image.originalUrl || image.thumbnailUrl;
}

function buildPlaceDraft(project: Project | null): PlaceDraft {
  const place = project ? project.place ?? resolveCityPlace(project.location ?? "") : null;
  return {
    location: project?.location ?? place?.city ?? "",
    city: place?.city ?? "",
    region: place?.region ?? "",
    country: place?.country ?? "中国",
    latitude: place ? String(place.latitude) : "",
    longitude: place ? String(place.longitude) : "",
  };
}

function parseCoordinate(value: string) {
  const normalized = value.trim();
  if (!normalized) return Number.NaN;
  return Number(normalized);
}

export function MemoryMapView({
  projects,
  busy,
  onSelectProject,
  onSaveProjectPlace,
  onBack,
}: MemoryMapViewProps) {
  const albumProjects = useMemo(() => projects.filter((project) => Boolean(project.album)), [projects]);
  const locatedProjects = useMemo<LocatedProject[]>(
    () =>
      albumProjects
        .map((project) => {
          const place = resolveProjectPlace(project);
          return place ? { project, place } : null;
        })
        .filter((item): item is LocatedProject => Boolean(item)),
    [albumProjects],
  );

  const cityGroups = useMemo<CityGroup[]>(() => {
    const groups = new Map<string, CityGroup>();
    locatedProjects.forEach(({ project, place }) => {
      const key = cityKey(place);
      const group = groups.get(key);
      if (group) {
        group.projects.push(project);
        group.imageCount += project.images.length;
        group.firstVisitedAt = Math.min(group.firstVisitedAt, timeValue(project.createdAt));
        return;
      }
      groups.set(key, {
        key,
        place,
        projects: [project],
        firstVisitedAt: timeValue(project.createdAt),
        imageCount: project.images.length,
      });
    });
    return Array.from(groups.values())
      .map((group) => ({
        ...group,
        projects: [...group.projects].sort((a, b) => timeValue(a.createdAt) - timeValue(b.createdAt)),
      }))
      .sort((a, b) => a.firstVisitedAt - b.firstVisitedAt);
  }, [locatedProjects]);

  const [activeCityKey, setActiveCityKey] = useState("");
  const [editingProjectId, setEditingProjectId] = useState("");
  const [draft, setDraft] = useState<PlaceDraft>(() => buildPlaceDraft(null));
  const [placeError, setPlaceError] = useState("");
  const [selectedStyleId, setSelectedStyleId] = useState('bright');
  const mapRef = useRef<MapRef>(null);

  useEffect(() => {
    if (cityGroups.length === 0) {
      setActiveCityKey("");
      return;
    }
    if (!cityGroups.some((group) => group.key === activeCityKey)) {
      setActiveCityKey(cityGroups[0].key);
    }
  }, [activeCityKey, cityGroups]);

  const activeCity = cityGroups.find((group) => group.key === activeCityKey) ?? cityGroups[0] ?? null;
  const editingProject = projects.find((project) => project.id === editingProjectId) ?? null;
  const autoPlace = editingProject ? resolveCityPlace(editingProject.location ?? "") : null;
  const isSavingPlace = editingProject ? busy === `place:${editingProject.id}` : false;
  const memoryImageCount = locatedProjects.reduce((total, item) => total + item.project.images.length, 0);

  const openPlaceEditor = (project: Project) => {
    setEditingProjectId(project.id);
    setDraft(buildPlaceDraft(project));
    setPlaceError("");
  };

  const closePlaceEditor = () => {
    if (isSavingPlace) return;
    setEditingProjectId("");
    setPlaceError("");
  };

  const selectCityMarker = useCallback((group: CityGroup) => {
    setActiveCityKey(group.key);
    mapRef.current?.flyTo({
      center: [group.place.longitude, group.place.latitude],
      zoom: 6,
      duration: 1000,
    });
  }, []);

  const saveAutoPlace = async () => {
    if (!editingProject || !autoPlace) return;
    setPlaceError("");
    try {
      await onSaveProjectPlace(editingProject.id, {
        location: editingProject.location?.trim() || autoPlace.city,
        place: autoPlace,
      });
      closePlaceEditor();
    } catch (err) {
      setPlaceError(err instanceof Error ? err.message : "地点保存失败");
    }
  };

  const saveManualPlace = async () => {
    if (!editingProject) return;
    const latitude = parseCoordinate(draft.latitude);
    const longitude = parseCoordinate(draft.longitude);
    if (!draft.city.trim()) {
      setPlaceError("城市不能为空。");
      return;
    }
    if (!draft.country.trim()) {
      setPlaceError("国家不能为空。");
      return;
    }
    if (!Number.isFinite(latitude) || latitude < -90 || latitude > 90) {
      setPlaceError("纬度需要在 -90 到 90 之间。");
      return;
    }
    if (!Number.isFinite(longitude) || longitude < -180 || longitude > 180) {
      setPlaceError("经度需要在 -180 到 180 之间。");
      return;
    }

    const place: ProjectPlace = {
      city: draft.city.trim(),
      region: draft.region.trim() || undefined,
      country: draft.country.trim(),
      latitude,
      longitude,
      source: "manual",
      confidence: 1,
    };
    setPlaceError("");
    try {
      await onSaveProjectPlace(editingProject.id, {
        location: draft.location.trim() || place.city,
        place,
      });
      closePlaceEditor();
    } catch (err) {
      setPlaceError(err instanceof Error ? err.message : "地点保存失败");
    }
  };

  return (
    <div className="memory-map-view">
      {/* Floating header with back button */}
      <header className="memory-map-header">
        <button type="button" className="memory-map-back" onClick={onBack}>
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <line x1="19" y1="12" x2="5" y2="12"/>
            <polyline points="12 19 5 12 12 5"/>
          </svg>
          <span>返回</span>
        </button>
        <div className="memory-map-title">
          <MapPin size={18} />
          <h1>集忆地图</h1>
        </div>
        <div className="memory-map-stats-inline">
          <span>{formatCount(cityGroups.length)} 城市</span>
          <span className="stat-divider">·</span>
          <span>{formatCount(locatedProjects.length)} 相册</span>
          <span className="stat-divider">·</span>
          <span>{formatCount(memoryImageCount)} 照片</span>
        </div>
      </header>

      {/* Main map area */}
      <div className="memory-map-stage" aria-label="集忆城市地图">
        {cityGroups.length === 0 ? (
          <div className="memory-map-empty">
            <div className="empty-icon">
              <MapPin size={48} strokeWidth={1.5} />
            </div>
            <h2>还没有可以入图的相册</h2>
            <p>生成相册并补充城市后，这里会出现第一枚地点标记。</p>
          </div>
        ) : null}

        {/* MapLibre GL Map */}
        <MapLibreMap
          ref={mapRef}
          initialViewState={DEFAULT_VIEW}
          mapStyle={MAP_STYLES.find(s => s.id === selectedStyleId)?.url || MAP_STYLES[0].url}
          style={{ width: '100%', height: '100%' }}
          onClick={() => setActiveCityKey("")}
          onLoad={(e) => {
            const map = e.target;

            // Apply warm theme styling for the 'warm' style
            if (selectedStyleId === 'warm') {
              try {
                // Warm land colors
                if (map.getLayer('land')) {
                  map.setPaintProperty('land', 'background-color', '#f5f0e8');
                }

                // Soft water colors
                if (map.getLayer('water')) {
                  map.setPaintProperty('water', 'fill-color', '#d4e4ed');
                }

                // Enhanced admin boundaries (provinces)
                const boundaryLayers = map.getStyle().layers || [];
                boundaryLayers.forEach(layer => {
                  if (layer.id.includes('boundary') || layer.id.includes('admin')) {
                    if (layer.id.includes('admin-1') || layer.id.includes('province')) {
                      // Province boundaries - make them visible
                      map.setPaintProperty(layer.id, 'line-color', '#8b7355');
                      map.setPaintProperty(layer.id, 'line-width', 1.2);
                      map.setPaintProperty(layer.id, 'line-opacity', 0.6);
                    } else if (layer.id.includes('admin-2') || layer.id.includes('country')) {
                      // Country boundaries
                      map.setPaintProperty(layer.id, 'line-color', '#6b5d4f');
                      map.setPaintProperty(layer.id, 'line-width', 1.5);
                      map.setPaintProperty(layer.id, 'line-opacity', 0.8);
                    }
                  }
                });
              } catch (err) {
                console.warn('Failed to apply warm theme styling:', err);
              }
            }
          }}
        >
          <NavigationControl position="top-left" />

          {/* Style Switcher */}
          <div className="map-style-switcher">
            {MAP_STYLES.map((style) => (
              <button
                key={style.id}
                type="button"
                className={`style-btn ${selectedStyleId === style.id ? 'active' : ''}`}
                onClick={() => setSelectedStyleId(style.id)}
                title={style.name}
              >
                {style.name}
              </button>
            ))}
          </div>

          {cityGroups.map((group) => (
            <Marker
              key={group.key}
              longitude={group.place.longitude}
              latitude={group.place.latitude}
              anchor="center"
              onClick={(e) => {
                e.originalEvent.stopPropagation();
                selectCityMarker(group);
              }}
            >
              <div className={`map-pin ${group.key === activeCityKey ? 'active' : ''}`}>
                <span className="map-pin-count">{group.projects.length}</span>
              </div>
              <span className="map-pin-label">{group.place.city}</span>
            </Marker>
          ))}
        </MapLibreMap>
      </div>

      {/* Floating sidebar */}
      {cityGroups.length > 0 && (
        <aside className="memory-map-sidebar">
          {/* City list */}
          <section className="sidebar-section">
            <div className="section-header">
              <Route size={16} />
              <h2>城市记忆</h2>
              <span className="section-count">{cityGroups.length}</span>
            </div>
            <div className="city-list">
              {cityGroups.map((group) => (
                <button
                  key={group.key}
                  type="button"
                  className="city-card"
                  data-active={group.key === activeCity?.key}
                  onClick={() => selectCityMarker(group)}
                >
                  <div className="city-info">
                    <strong>{formatPlaceLabel(group.place)}</strong>
                    <span>{formatCount(group.projects.length)} 相册 · {formatCount(group.imageCount)} 照片</span>
                  </div>
                  <div className="city-arrow">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                      <polyline points="9 18 15 12 9 6"/>
                    </svg>
                  </div>
                </button>
              ))}
            </div>
          </section>

          {/* Album list for selected city */}
          {activeCity && (
            <section className="sidebar-section album-section">
              <div className="section-header">
                <BookImage size={16} />
                <h2>{activeCity.place.city}</h2>
                <span className="section-count">{activeCity.projects.length}</span>
              </div>
              <div className="album-list">
                {activeCity.projects.map((project) => {
                  const cover = projectCoverImage(project);
                  return (
                    <article key={project.id} className="album-card">
                      <button
                        type="button"
                        className="album-cover"
                        onClick={() => onSelectProject(project.id)}
                        aria-label={`打开 ${project.title}`}
                      >
                        {cover ? (
                          <img src={imageDisplayUrl(cover)} alt={project.title} />
                        ) : (
                          <BookImage size={24} />
                        )}
                      </button>
                      <div className="album-info">
                        <strong>{project.album?.title || project.title}</strong>
                        <span>{project.location || formatPlaceLabel(activeCity.place)}</span>
                        <small>
                          {formatDate(project.createdAt)} · {formatCount(project.album?.pages.length ?? 0)} 页 ·{" "}
                          {formatCount(project.images.length)} 张
                        </small>
                      </div>
                      <button
                        type="button"
                        className="album-edit"
                        onClick={() => openPlaceEditor(project)}
                        aria-label="编辑地点"
                      >
                        <Pencil size={14} />
                      </button>
                    </article>
                  );
                })}
              </div>
            </section>
          )}
        </aside>
      )}

      {/* Place editor dialog */}
      <DialogShell
        open={Boolean(editingProject)}
        onClose={closePlaceEditor}
        rootClassName="memory-map-dialog"
        backdropClassName="memory-map-dialog-backdrop"
        panelClassName="memory-map-dialog-panel"
        ariaLabel="编辑相册地点"
        zIndex={65}
      >
        <div className="dialog-header">
          <div>
            <h3>编辑相册地点</h3>
            <p>{editingProject?.album?.title || editingProject?.title}</p>
          </div>
          <button type="button" className="dialog-close" onClick={closePlaceEditor} aria-label="关闭">
            <X size={20} />
          </button>
        </div>

        <div className="dialog-body">
          <label className="dialog-field">
            <span>地点显示名</span>
            <input
              value={draft.location}
              onChange={(event) => setDraft((current) => ({ ...current, location: event.target.value }))}
              placeholder="例如 上海、京都、巴黎"
            />
          </label>
          <div className="dialog-row">
            <label className="dialog-field">
              <span>城市</span>
              <input
                value={draft.city}
                onChange={(event) => setDraft((current) => ({ ...current, city: event.target.value }))}
                placeholder="城市"
              />
            </label>
            <label className="dialog-field">
              <span>地区/省份</span>
              <input
                value={draft.region}
                onChange={(event) => setDraft((current) => ({ ...current, region: event.target.value }))}
                placeholder="地区或省份"
              />
            </label>
          </div>
          <label className="dialog-field">
            <span>国家</span>
            <input
              value={draft.country}
              onChange={(event) => setDraft((current) => ({ ...current, country: event.target.value }))}
              placeholder="国家"
            />
          </label>
          <div className="dialog-row">
            <label className="dialog-field">
              <span>纬度</span>
              <input
                value={draft.latitude}
                onChange={(event) => setDraft((current) => ({ ...current, latitude: event.target.value }))}
                placeholder="31.2304"
              />
            </label>
            <label className="dialog-field">
              <span>经度</span>
              <input
                value={draft.longitude}
                onChange={(event) => setDraft((current) => ({ ...current, longitude: event.target.value }))}
                placeholder="121.4737"
              />
            </label>
          </div>

          {autoPlace ? (
            <button type="button" className="dialog-auto-btn" onClick={saveAutoPlace} disabled={isSavingPlace}>
              <LocateFixed size={16} />
              <span>使用自动匹配：{formatPlaceLabel(autoPlace)}</span>
            </button>
          ) : null}

          {placeError ? (
            <div className="dialog-error">
              {placeError}
            </div>
          ) : null}
        </div>

        <div className="dialog-footer">
          <button type="button" className="dialog-btn-secondary" onClick={closePlaceEditor} disabled={isSavingPlace}>
            取消
          </button>
          <button type="button" className="dialog-btn-primary" onClick={saveManualPlace} disabled={isSavingPlace}>
            {isSavingPlace ? "保存中..." : (
              <>
                <Check size={16} />
                <span>保存坐标</span>
              </>
            )}
          </button>
        </div>
      </DialogShell>
    </div>
  );
}
