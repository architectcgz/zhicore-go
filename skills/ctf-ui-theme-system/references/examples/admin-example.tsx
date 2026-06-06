import React, { useState, useMemo, useEffect } from 'react';
import { 
  Book, CheckCircle, Edit3, Search, Filter, MoreHorizontal, Plus, Database,
  Calendar, Users, Activity, ChevronDown, ChevronLeft, ChevronRight, Tag,
  Zap, Eye, FileSearch, Layers, BarChart3, Fingerprint, X, RotateCcw,
  ArrowUpNarrowWide, ArrowDownWideNarrow, SortAsc, Pencil, Copy, Download,
  Trash2, Power, LayoutDashboard, Swords, Settings, Bell, Box, Moon, Sun,
  Palette, Info
} from 'lucide-react';

const App = () => {
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  
  // 侧边栏与主题状态
  const [isCollapsed, setIsCollapsed] = useState(false);
  const [activeMenu, setActiveMenu] = useState('Contest');
  const [isDarkMode, setIsDarkMode] = useState(false);
  
  const topNavTabs = ['环境管理', '测试机分配', '流量监控', '大屏投射', '公告管理', '排行榜', '作弊检测', '导出数据'];
  const [activeSubTab, setActiveSubTab] = useState('环境管理');

  const [searchQuery, setSearchQuery] = useState('');
  const [isFilterOpen, setIsFilterOpen] = useState(false);
  const [isSortOpen, setIsSortOpen] = useState(false);
  const [activeMenuId, setActiveMenuId] = useState(null);
  
  const [filterCategory, setFilterCategory] = useState('全部');
  const [filterDifficulty, setFilterDifficulty] = useState('全部');
  const [sortConfig, setSortConfig] = useState({ key: 'updateTime', order: 'desc', label: '最近更新' });

  useEffect(() => {
    const handleClickOutside = () => {
      setActiveMenuId(null);
      setIsSortOpen(false);
      setIsFilterOpen(false);
    };
    window.addEventListener('click', handleClickOutside);
    return () => window.removeEventListener('click', handleClickOutside);
  }, []);

  const menuItems = [
    { id: 'Dashboard', label: '仪表盘', icon: <LayoutDashboard size={18} /> },
    { id: 'Contest', label: '赛事运维', icon: <Activity size={18} /> },
    { id: 'Challenge', label: '题目管理', icon: <Swords size={18} /> },
    { id: 'Config', label: '系统配置', icon: <Settings size={18} /> },
    { id: 'Announcement', label: '全局公告', icon: <Bell size={18} /> },
    { id: 'Container', label: '容器实例', icon: <Box size={18} /> },
  ];

  const stats = [
    { label: '题目总数', value: 4, icon: <Book size={16} />, trend: '+0%', detail: '题目资源总计', colorClass: 'text-blue-500' },
    { label: '已发布', value: 3, icon: <CheckCircle size={16} />, trend: '75%', detail: '线上公开题目', colorClass: 'text-emerald-500' },
    { label: '环境运行', value: 2, icon: <Zap size={16} />, trend: '正常', detail: '运行状态良好', colorClass: 'text-purple-500' },
    { label: '待处理', value: 1, icon: <Edit3 size={16} />, trend: '待办', detail: '需审核题目', colorClass: 'text-orange-500' },
  ];

  const challenges = [
    { id: 1, uuid: 'CHALLENGE-WEB-N01', title: '内部笔记下载器', category: 'Web', difficulty: '简单', points: 100, status: '已发布', updateTime: '2024-03-20', users: 124, health: '检查通过' },
    { id: 2, uuid: 'CHALLENGE-WEB-N02', title: 'Web-01 源码审计：双层伪装', category: 'Web', difficulty: '简单', points: 120, status: '已发布', updateTime: '2024-03-19', users: 85, health: '检查通过' },
    { id: 3, uuid: 'CHALLENGE-PWN-N01', title: '格式化字符串漏洞利用初探', category: 'Pwn', difficulty: '中等', points: 250, status: '草稿', updateTime: '2024-04-10', users: 0, health: '待检查' },
    { id: 4, uuid: 'CHALLENGE-MISC-099', title: '大容量附件隐写分析挑战赛题目', category: 'Misc', difficulty: '困难', points: 500, status: '已发布', updateTime: '2024-04-14', users: 5, health: '检查通过' },
  ];

  const sortOptions = [
    { key: 'updateTime', order: 'desc', label: '最近更新', icon: <Calendar size={14} /> },
    { key: 'points', order: 'desc', label: '分值由高到低', icon: <ArrowDownWideNarrow size={14} /> },
    { key: 'points', order: 'asc', label: '分值由低到高', icon: <ArrowUpNarrowWide size={14} /> },
    { key: 'title', order: 'asc', label: '标题 A-Z', icon: <SortAsc size={14} /> },
  ];

  const processedChallenges = useMemo(() => {
    let result = challenges.filter(c => {
      const matchSearch = c.title.toLowerCase().includes(searchQuery.toLowerCase()) || c.uuid.toLowerCase().includes(searchQuery.toLowerCase());
      const matchCategory = filterCategory === '全部' || c.category === filterCategory;
      const matchDifficulty = filterDifficulty === '全部' || c.difficulty === filterDifficulty;
      return matchSearch && matchCategory && matchDifficulty;
    });
    result.sort((a, b) => {
      const valA = a[sortConfig.key];
      const valB = b[sortConfig.key];
      return sortConfig.order === 'asc' ? (valA > valB ? 1 : -1) : (valA < valB ? 1 : -1);
    });
    return result;
  }, [searchQuery, filterCategory, filterDifficulty, sortConfig]);

  const resetFilters = () => {
    setSearchQuery('');
    setFilterCategory('全部');
    setFilterDifficulty('全部');
    setIsFilterOpen(false);
  };

  const renderSidebar = () => (
    <aside className={`${isCollapsed ? 'w-20' : 'w-64'} bg-white flex flex-col h-screen shrink-0 border-r border-slate-200 transition-all duration-300 relative z-30`}>
      <button onClick={() => setIsCollapsed(!isCollapsed)} className="absolute -right-3.5 top-6 bg-white border border-slate-200 rounded-full p-1.5 text-slate-400 hover:text-blue-600 hover:border-blue-300 hover:shadow-md shadow-sm z-50 transition-all cursor-pointer">
        {isCollapsed ? <ChevronRight size={14} /> : <ChevronLeft size={14} />}
      </button>
      <div className="h-16 flex items-center px-5 border-b border-slate-100 overflow-hidden whitespace-nowrap">
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 shrink-0 bg-slate-900 rounded-lg flex items-center justify-center shadow-sm"><Database className="text-white" size={16} /></div>
          <span className={`font-black text-lg tracking-tight uppercase text-slate-900 transition-opacity duration-200 ${isCollapsed ? 'opacity-0' : 'opacity-100'}`}>Challenge<span className="text-blue-600">Ops</span></span>
        </div>
      </div>
      <div className={`px-6 py-5 overflow-hidden whitespace-nowrap transition-all duration-200 ${isCollapsed ? 'opacity-0 h-0 p-0' : 'opacity-100 h-14'}`}><span className="text-[10px] font-black text-slate-400 uppercase tracking-widest">Main Navigation</span></div>
      <nav className={`flex-1 space-y-2 overflow-x-hidden ${isCollapsed ? 'px-3 pt-4' : 'px-4'}`}>
        {menuItems.map(item => (
          <button key={item.id} onClick={() => setActiveMenu(item.id)} title={isCollapsed ? item.label : ''} className={`w-full flex items-center gap-3 py-2.5 rounded-xl text-sm transition-all overflow-hidden ${activeMenu === item.id ? 'bg-blue-50 text-blue-600 font-bold shadow-sm border border-blue-100/50' : 'text-slate-500 hover:text-slate-900 hover:bg-slate-50 font-medium border border-transparent'} ${isCollapsed ? 'px-0 justify-center' : 'px-3'}`}>
            <div className={`shrink-0 ${activeMenu === item.id ? 'text-blue-500' : 'text-slate-400'}`}>{item.icon}</div>
            <span className={`transition-opacity duration-200 whitespace-nowrap ${isCollapsed ? 'opacity-0 hidden' : 'opacity-100'}`}>{item.label}</span>
          </button>
        ))}
      </nav>
    </aside>
  );

  return (
    <div className="flex h-screen bg-[#f8fafc] overflow-hidden">
      {renderSidebar()}

      <main className="flex-1 flex flex-col min-w-0 overflow-y-auto">
        <header className="h-16 bg-white border-b border-slate-200 flex items-center justify-between px-8 sticky top-0 z-20 shrink-0">
          <div className="flex items-center text-sm font-bold text-slate-500">
            <span className="text-slate-400">Workspace</span>
            <span className="mx-2 text-slate-300">/</span>
            <span className="text-slate-900 font-black">{menuItems.find(m => m.id === activeMenu)?.label}</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="flex items-center gap-1">
              <button className="w-9 h-9 relative flex items-center justify-center rounded-full text-slate-400 hover:text-slate-600 hover:bg-slate-100 transition-colors" title="系统通知">
                <Bell size={18} />
                <span className="absolute top-2.5 right-2.5 w-1.5 h-1.5 bg-red-500 rounded-full border border-white"></span>
              </button>
              <button onClick={() => setIsDarkMode(!isDarkMode)} className="w-9 h-9 flex items-center justify-center rounded-full text-slate-400 hover:text-slate-600 hover:bg-slate-100 transition-colors" title="切换模式">
                {isDarkMode ? <Sun size={18} /> : <Moon size={18} />}
              </button>
              <button className="w-9 h-9 flex items-center justify-center rounded-full text-slate-400 hover:text-slate-600 hover:bg-slate-100 transition-colors" title="配色方案">
                <Palette size={18} />
              </button>
            </div>
            <div className="w-[1px] h-4 bg-slate-200 mx-2"></div>
            <div className="flex items-center gap-2 cursor-pointer hover:bg-slate-50 p-1 pr-3 rounded-full transition-colors border border-transparent hover:border-slate-100">
              <div className="w-7 h-7 rounded-full bg-slate-100 border border-slate-200 flex items-center justify-center overflow-hidden"><Users size={12} className="text-slate-400" /></div>
              <div className="flex flex-col"><span className="text-xs font-black text-slate-700 leading-none">Admin</span><span className="text-[9px] font-bold text-slate-400 leading-none mt-1 uppercase">Root</span></div>
              <ChevronDown size={14} className="text-slate-400" />
            </div>
          </div>
        </header>

        <div className="p-8 max-w-[1500px] mx-auto w-full">
          
          <div className="flex flex-col sm:flex-row sm:justify-between sm:items-end mb-6 gap-4">
            <div>
              <h1 className="text-2xl font-black text-slate-900 tracking-tight">2026 AWD 决赛运维台</h1>
              <p className="text-slate-400 text-[11px] font-bold uppercase tracking-wider mt-1 font-medium italic">Operations / Contest Management</p>
            </div>
            <div className="flex gap-2">
              <button className="bg-white border border-slate-200 text-slate-600 px-4 py-2 rounded-xl font-bold text-xs hover:bg-slate-50 transition-all flex items-center gap-1.5 shadow-sm"><FileSearch size={14} /> 审计日志</button>
              <button className="bg-blue-600 text-white px-5 py-2 rounded-xl font-bold text-xs shadow-lg shadow-blue-100 hover:bg-blue-700 transition-all flex items-center gap-1.5 active:scale-95"><Plus size={16} /> 导入资源包</button>
            </div>
          </div>

          <div className="flex items-center gap-8 border-b border-slate-200 mb-8 overflow-x-auto hide-scrollbar">
            {topNavTabs.map(tab => (
              <button key={tab} onClick={() => setActiveSubTab(tab)} className={`pb-3 text-sm font-bold whitespace-nowrap transition-colors relative flex-shrink-0 ${activeSubTab === tab ? 'text-blue-600' : 'text-slate-500 hover:text-slate-900'}`}>
                {tab}
                {activeSubTab === tab && <span className="absolute bottom-0 left-0 w-full h-0.5 bg-blue-600 rounded-t-full shadow-[0_-2px_8px_rgba(37,99,235,0.5)]"></span>}
              </button>
            ))}
          </div>

          {/* 顶部的统计大卡片 */}
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-10">
            {stats.map((s) => (
              <div key={s.label} className="bg-white border border-slate-200 rounded-xl p-4 shadow-sm hover:border-blue-400 transition-all group relative overflow-hidden">
                <div className="absolute top-0 right-0 w-12 h-12 bg-slate-50 rounded-bl-full -mr-6 -mt-6 group-hover:bg-blue-50 transition-colors"></div>
                <div className="flex flex-col h-full relative z-10">
                  <div className="mb-auto flex items-start justify-between text-[12px] font-bold uppercase tracking-[0.08em] text-slate-400 transition-colors group-hover:text-blue-500">
                    <span>{s.label}</span>{s.icon}
                  </div>
                  <div className="flex items-end justify-between mt-4">
                    <div className="flex flex-col">
                      <h3 className="font-mono text-[2.5rem] font-black leading-none tracking-[-0.06em] text-slate-900">{s.value.toString().padStart(2, '0')}</h3>
                      <p className="mt-2 text-[12px] font-bold tracking-[0.04em] text-slate-500">{s.detail}</p>
                    </div>
                    <div className="text-right font-bold"><span className="rounded-md border border-emerald-100 bg-emerald-50 px-2 py-0.5 text-[11px] font-bold text-emerald-500">{s.trend}</span></div>
                  </div>
                </div>
              </div>
            ))}
          </div>

          {/* ========================================================= */}
          {/* 标准完整列表区：标题 + 轻工具栏 + 平铺列表 + 贴底分页 */}
          {/* ========================================================= */}

          <section className="relative bg-transparent">
            <header className="flex flex-wrap items-end justify-between gap-4 mb-5">
              <div>
                <div className="text-[11px] font-black uppercase tracking-[0.24em] text-slate-400">
                  Class Directory
                </div>
                <h2 className="mt-1 text-[1.6rem] font-black tracking-tight text-slate-900">
                  班级目录
                </h2>
              </div>
            </header>
            
            {/* 1. 轻工具栏 (筛选与搜索) */}
            <div className="relative mb-6 z-10 flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
              <div className="flex w-full flex-wrap items-center gap-2 lg:w-auto">
                <div className="relative flex-1 sm:flex-none">
                  <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" size={14} />
                  <input 
                    type="text" value={searchQuery} onChange={(e) => setSearchQuery(e.target.value)}
                    placeholder="检索班级编号或名称..." 
                    className="h-10 w-full rounded-xl border border-slate-300/90 bg-white pl-9 pr-4 text-[14px] font-medium text-slate-700 shadow-[0_1px_2px_rgba(15,23,42,0.04)] outline-none transition-all hover:border-slate-400 focus:border-blue-400 focus:ring-2 focus:ring-blue-500/10 sm:w-80"
                  />
                </div>
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    setIsFilterOpen(!isFilterOpen);
                    setIsSortOpen(false);
                  }}
                  className={`inline-flex h-10 items-center gap-1.5 rounded-xl border px-4 text-[13px] font-bold shadow-[0_1px_2px_rgba(15,23,42,0.04)] transition-all ${
                    isFilterOpen
                      ? 'border-blue-200 bg-blue-50 text-blue-600'
                      : 'border-slate-300/90 bg-white text-slate-700 hover:border-slate-400 hover:text-blue-600'
                  }`}
                >
                  <Filter size={14} /> 筛选
                </button>
              </div>

              <div className="flex flex-wrap items-center gap-3 text-[11px] font-bold uppercase tracking-tight text-slate-400">
                <div className="relative">
                  <span className="mr-1">排序: </span>
                  <button onClick={(e) => { e.stopPropagation(); setIsSortOpen(!isSortOpen); setIsFilterOpen(false); }} className="inline-flex h-10 items-center gap-1 rounded-xl border border-slate-300/90 bg-white px-4 text-[13px] text-slate-700 shadow-[0_1px_2px_rgba(15,23,42,0.04)] transition-all hover:border-slate-400 hover:text-blue-600">
                    <span className="font-black tracking-wider">{sortConfig.label}</span>
                    <ChevronDown size={12} className={`transition-transform ${isSortOpen ? 'rotate-180' : ''}`} />
                  </button>
                  {isSortOpen && (
                    <div className="absolute top-full right-0 mt-2 w-52 bg-white border border-slate-200 rounded-xl shadow-2xl z-50 overflow-hidden animate-in fade-in zoom-in-95 duration-100 font-bold">
                      <div className="bg-slate-50/50 border-b border-slate-100 px-4 py-2 text-[10px] font-black uppercase text-slate-400">Sort Strategy</div>
                      <div className="py-1">
                        {sortOptions.map((opt) => (
                          <button key={opt.label} onClick={() => { setSortConfig(opt); setIsSortOpen(false); }} className="w-full flex items-center justify-between px-4 py-2.5 text-[13px] font-bold transition-colors hover:bg-slate-50 text-slate-600">
                            <div className="flex items-center gap-2">{opt.icon}{opt.label}</div>
                          </button>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
                <div className="inline-flex h-10 items-center rounded-xl border border-slate-300/90 bg-white px-4 text-[13px] text-slate-500 shadow-[0_1px_2px_rgba(15,23,42,0.04)]">
                  共 <span className="mx-1 font-mono font-black text-slate-900">{processedChallenges.length}</span> 个班级
                </div>
              </div>
            </div>

            {/* 高级筛选浮层。输入与选择即生效，浮层只负责承载低频筛选。 */}
            {isFilterOpen && (
              <div className="absolute left-0 z-50 mt-2 w-full max-w-sm rounded-2xl border border-slate-200 bg-white p-5 font-bold shadow-2xl animate-in slide-in-from-top-2 duration-200">
                <div className="flex justify-between items-center mb-4 pb-2 border-b border-slate-50">
                  <div>
                    <div className="text-[11px] font-black uppercase tracking-[0.22em] text-slate-400">Filter Stack</div>
                    <span className="mt-1 block text-[15px] font-black text-slate-900">高级筛选</span>
                  </div>
                  <button onClick={() => setIsFilterOpen(false)} className="text-slate-400 hover:text-slate-900"><X size={16} /></button>
                </div>
                <div className="space-y-4">
                  <div>
                    <label className="mb-2 block text-[11px] font-black uppercase tracking-widest text-slate-400">班级分类</label>
                    <div className="flex flex-wrap gap-2">
                      {['全部', 'Web', 'Pwn', 'Reverse'].map(cat => (<button key={cat} onClick={() => setFilterCategory(cat)} className={`rounded-lg border px-3 py-1.5 text-[13px] font-bold transition-all ${filterCategory === cat ? 'border-blue-600 bg-blue-600 text-white' : 'border-slate-200 bg-white text-slate-600 hover:border-blue-300'}`}>{cat}</button>))}
                    </div>
                  </div>
                  <div>
                    <label className="mb-2 block text-[11px] font-black uppercase tracking-widest text-slate-400">班级状态</label>
                    <div className="flex flex-wrap gap-2">
                      {['全部', '简单', '中等', '困难'].map(diff => (<button key={diff} onClick={() => setFilterDifficulty(diff)} className={`rounded-lg border px-3 py-1.5 text-[13px] font-bold transition-all ${filterDifficulty === diff ? 'border-slate-900 bg-slate-900 text-white' : 'border-slate-200 bg-white text-slate-600 hover:border-slate-400'}`}>{diff}</button>))}
                    </div>
                  </div>
                </div>
                <div className="mt-6 flex justify-between gap-3 border-t border-slate-50 pt-4">
                  <button onClick={resetFilters} className="text-[12px] font-bold uppercase tracking-widest text-slate-400 transition-colors hover:text-slate-900">清空条件</button>
                  <button onClick={() => setIsFilterOpen(false)} className="text-[12px] font-bold uppercase tracking-widest text-slate-500 transition-colors hover:text-slate-900">收起筛选</button>
                </div>
              </div>
            )}

            {/* 2. 平铺列表主体 */}
            <div className="overflow-visible rounded-[1.35rem] border border-slate-200/80 bg-white/80 px-4 pb-2 pt-1 font-bold">
              <table className="w-full table-fixed border-collapse text-left overflow-visible">
                <thead className="border-b border-slate-200/50 text-slate-400">
                  <tr>
                    <th className="w-[16%] min-w-[120px] px-2 py-4 text-[11px] font-black uppercase tracking-widest">班级编号</th>
                    <th className="w-[34%] min-w-[220px] px-2 py-4 text-[11px] font-black uppercase tracking-widest">班级名称</th>
                    <th className="w-[14%] px-2 py-4 text-center text-[11px] font-black uppercase tracking-widest">学生数</th>
                    <th className="w-[16%] px-2 py-4 text-center text-[11px] font-black uppercase tracking-widest">状态</th>
                    <th className="w-[20%] px-2 py-4 text-right text-[11px] font-black uppercase tracking-widest">操作</th>
                  </tr>
                </thead>
                <tbody className="text-sm overflow-visible">
                  {processedChallenges.map((c, index) => {
                    const isLastRows = index >= processedChallenges.length - 2 && processedChallenges.length > 2;
                    return (
                      <tr key={c.id} className={`group transition-all hover:bg-slate-50 ${index === processedChallenges.length - 1 ? 'border-b border-transparent' : 'border-b border-slate-200/55'}`}>
                        <td className="px-2 py-3.5 font-mono text-[12px] font-bold uppercase tracking-[0.08em] text-slate-500" title={c.uuid}>{c.uuid}</td>
                        <td className="truncate px-2 py-3.5 text-[17px] font-bold text-slate-900 transition-colors group-hover:text-blue-600" title={c.title}>{c.title}</td>
                        <td className="px-2 py-3.5 text-center font-mono text-[15px] font-black tracking-tight text-slate-900">{c.users}</td>
                        <td className="px-2 py-3.5 text-center">
                          <span className={`inline-flex min-w-[82px] items-center justify-center rounded-full border px-3 py-1 text-[12px] font-bold ${c.users > 0 ? 'border-emerald-200 bg-emerald-50 text-emerald-700' : 'border-slate-200 bg-slate-50 text-slate-500'}`}>
                            {c.users > 0 ? '可查看' : '待入班'}
                          </span>
                        </td>
                        <td className="py-3.5 text-right relative overflow-visible px-2">
                          <div className="flex items-center justify-end gap-1.5">
                            <button className="inline-flex min-h-[36px] items-center gap-1 rounded-[10px] bg-blue-50 px-3.5 py-1.5 text-[12px] font-black text-blue-600 transition-all hover:bg-blue-600 hover:text-white hover:shadow-md active:scale-95"><Eye size={12} /> 查看班级</button>
                            <div className="relative inline-block text-left">
                              <button onClick={(e) => { e.stopPropagation(); setActiveMenuId(activeMenuId === c.id ? null : c.id); }} className={`p-1.5 rounded-lg transition-all ${activeMenuId === c.id ? 'bg-slate-900 text-white shadow-lg' : 'text-slate-400 hover:text-slate-900 hover:bg-slate-100'}`}><MoreHorizontal size={14} /></button>
                              {activeMenuId === c.id && (
                                <div className={`absolute right-0 w-44 bg-white border border-slate-200 rounded-xl shadow-2xl z-[100] py-1 overflow-hidden animate-in fade-in zoom-in-95 duration-100 ${isLastRows ? 'bottom-full mb-2 origin-bottom-right' : 'top-full mt-2 origin-top-right'}`}>
                                  <div className="px-3 py-2 bg-slate-50/50 border-b border-slate-100 text-[10px] font-black text-slate-400 uppercase tracking-widest">Management</div>
                                  <button className="w-full flex items-center gap-2 px-4 py-2 text-[12px] font-bold text-slate-600 hover:bg-slate-50 hover:text-blue-600 transition-colors"><Pencil size={12} /> 编辑班级</button>
                                  <button className="w-full flex items-center gap-2 px-4 py-2 text-[12px] font-bold text-slate-600 hover:bg-slate-50 hover:text-blue-600 transition-colors"><Copy size={12} /> 复制链接</button>
                                  <button className="w-full flex items-center gap-2 px-4 py-2 text-[12px] font-bold text-slate-600 hover:bg-slate-50 hover:text-blue-600 transition-colors border-b border-slate-100"><Download size={12} /> 导出名单</button>
                                  <button className="w-full flex items-center gap-2 px-4 py-2 text-[12px] font-bold text-orange-500 hover:bg-orange-50 transition-colors"><Power size={12} /> 冻结班级</button>
                                  <button className="w-full flex items-center gap-2 px-4 py-2 text-[12px] font-black text-red-500 hover:bg-red-50 transition-colors"><Trash2 size={12} /> 删除班级</button>
                                </div>
                              )}
                            </div>
                          </div>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>

            {/* 3. 贴底分页 */}
            <div className="mt-6 flex flex-col items-center justify-between gap-4 text-[13px] font-medium text-slate-500 sm:flex-row">
              <div className="flex items-center gap-6 font-bold">
                <button className="group flex items-center gap-4 rounded-xl border border-slate-300/90 bg-white px-4 py-2 shadow-[0_1px_2px_rgba(15,23,42,0.04)] transition-all hover:border-slate-400 hover:text-blue-600">
                  <span className="font-bold text-slate-700">{pageSize} 条/页</span>
                  <ChevronDown size={14} className="text-slate-300 group-hover:text-blue-400" />
                </button>
              </div>
              <div className="flex items-center gap-6 font-bold">
                <div className="flex items-center gap-1.5 font-bold">
                  <button className="flex h-8 w-8 cursor-not-allowed items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-300 shadow-sm">
                    <ChevronLeft size={16} />
                  </button>
                  <button className="translate-y-[-1px] flex h-8 w-8 items-center justify-center rounded-lg bg-blue-600 font-black text-white shadow-md shadow-blue-200">
                    1
                  </button>
                  <button className="flex h-8 w-8 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-500 shadow-sm transition-all hover:text-blue-600">
                    <ChevronRight size={16} />
                  </button>
                </div>
              </div>
            </div>
            
          </section>
        </div>
      </main>
    </div>
  );
};

export default App;
