#pragma once
#include "TableHelp.h"

namespace Store
{
	struct stShop
	{
		stShop()
		{
			id = 0;
			desc = NULL;
			goldtype = 0;
		}

		~stShop()
		{
			delete[] desc;
		}

		typedef int32 KeyType;

		KeyType getKey()
		{
			return id;
		}

		int id; // 编号
		const char* desc; // 描述
		int goldtype; // 货币

		struct Goods
		{
			Goods()
			{
				itemid = 0;
				price = 0;
			}
			~Goods()
			{
			}

			int itemid; // 道具
			int price; // 价格
		};
		vector<Goods> goods;
	};

	inline Vek::Stream & operator >> ( Vek::Stream &data, stShop &row )
	{
		data >> row.id;
		data >> row.desc;
		data >> row.goldtype;

		ushort nSize = 0;
		data >> nSize;
		row.goods.resize( nSize );
		for( int k = 0; k < nSize; ++k )
		{
			data >> row.goods[k].itemid;
			data >> row.goods[k].price;
		}

		return data;
	}
}

class ShopReader : public TableReader<Store::stShop>, public Vek::Singleton<ShopReader>
{
	friend class Vek::Singleton<ShopReader>;
public:
	int InitializeX()
	{
		if( !LoadData( Store::GetStorePath("Shop_s.tbl").c_str() ) )
		{
			assert(false && "LoadData Shop_s.tbl, fail!");
			return 1;
		}
		return 0;
	}
private:
	ShopReader(){ InitializeX(); };
};
